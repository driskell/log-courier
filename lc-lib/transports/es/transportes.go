/*
 * Copyright 2012-2020 Jason Woods and contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package es

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/payload"
	"github.com/driskell/log-courier/lc-lib/transports"
)

var (
	globalTemplateLock sync.Mutex
)

// transportES implements a transport that sends over the ES HTTP protocol
type transportES struct {
	// Constructor
	ctx          context.Context
	shutdownFunc context.CancelFunc
	config       *TransportESFactory
	netConfig    *transports.Config
	finishOnFail bool
	pool         *addresspool.Pool
	eventChan    chan<- transports.Event

	// Internal
	// payloadMutex is so we can easily discard existing sendChan and its contents each time we reset
	payloadChan     chan *payload.Payload
	payloadMutex    sync.Mutex
	poolMutex       sync.Mutex
	wait            sync.WaitGroup
	nodeInfo        *nodeInfo
	maxMajorVersion int
}

// ReloadConfig returns true if the transport needs to be restarted in order
// for the new configuration to apply
func (t *transportES) ReloadConfig(netConfig *transports.Config, finishOnFail bool) bool {
	// Check if automatic retry should be enabled or not
	if t.finishOnFail != finishOnFail {
		return true
	}

	// Check other changes
	if t.netConfig.MaxPendingPayloads != netConfig.MaxPendingPayloads || t.netConfig.Timeout != netConfig.Timeout {
		return true
	}

	return false
}

// startController starts the controller
func (t *transportES) startController() {
	go t.controllerRoutine()
}

// controllerRoutine is the master routine which handles submission
func (t *transportES) controllerRoutine() {
	defer func() {
		// Wait for all routines to close
		t.wait.Wait()
		t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Finished)
	}()

	// Setup payload chan with max write count of pending payloads
	t.payloadMutex.Lock()
	t.payloadChan = make(chan *payload.Payload, t.netConfig.MaxPendingPayloads)
	t.payloadMutex.Unlock()

	if t.setupAssociation() {
		// Shutdown was requested
		return
	}

	t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Started)

	// Start secondary http routines
	for i := 1; i < t.config.Routines; i++ {
		t.wait.Add(1)
		go t.httpRoutine(i)
	}

	// Become the main http routine
	t.wait.Add(1)
	t.httpRoutine(0)
}

// setupAssociation gathers cluster information and installs templates
func (t *transportES) setupAssociation() bool {
	backoffName := fmt.Sprintf("%s Setup Retry", t.pool.Server())
	backoff := core.NewExpBackoff(backoffName, t.config.Retry, t.config.RetryMax)

	for {
		if err := t.populateNodeInfo(); err != nil {
			log.Errorf("[%s] Failed to fetch Elasticsearch node information: %s", t.pool.Server(), err)
		} else {
			if err := t.installTemplate(); err != nil {
				log.Errorf("[%s] Failed to install Elasticsearch index template: %s", t.pool.Server(), err)
			} else {
				return false
			}
		}

		if t.retryWait(backoff) {
			break
		}
	}

	// Shutdown
	return true
}

// populateNodeInfo populates the nodeInfo structure
func (t *transportES) populateNodeInfo() error {
	server, err := t.pool.Next()
	if err != nil {
		return err
	}

	httpRequest, err := http.NewRequestWithContext(t.ctx, "GET", fmt.Sprintf("http://%s/_nodes/http", server.String()), nil)
	if err != nil {
		return err
	}

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return err
	}
	defer func() {
		bufio.NewReader(httpResponse.Body).WriteTo(ioutil.Discard)
		httpResponse.Body.Close()
	}()
	if httpResponse.StatusCode != 200 {
		body, _ := ioutil.ReadAll(httpResponse.Body)
		return fmt.Errorf("Unexpected status: %s [Body: %s]", httpResponse.Status, body)
	}

	decoder := json.NewDecoder(httpResponse.Body)
	if err := decoder.Decode(&t.nodeInfo); err != nil {
		return err
	}

	t.maxMajorVersion, err = t.nodeInfo.MaxMajorVersion()
	if err != nil {
		return fmt.Errorf("Failed to calculate maximum version number for cluster: %s", err)
	}

	log.Info("[%s] Successfully retrieved Elasticsearch node information (major version: %d)", t.pool.Server(), t.maxMajorVersion)

	return nil
}

// installTemplate checks if template installation is needed
func (t *transportES) installTemplate() error {
	server, err := t.pool.Next()
	if err != nil {
		return err
	}

	name := "logstash"
	var (
		templateReader io.Reader
		templateLen    int
	)
	if t.config.template != nil {
		templateReader = bytes.NewReader(t.config.template)
		templateLen = len(t.config.template)
	} else {
		var template string
		switch t.maxMajorVersion {
		case 8:
			template = esTemplate8
			break
		case 7:
			template = esTemplate7
			break
		case 6:
			template = esTemplate6
			break
		case 5:
			template = esTemplate5
			break
		default:
			return fmt.Errorf("Elasticsearch major version %d is unsupported", t.maxMajorVersion)
		}
		templateReader = strings.NewReader(template)
		templateLen = len(template)
	}

	installed, err := t.checkTemplate(server, name)
	if err != nil {
		return err
	} else if installed {
		return nil
	}

	httpRequest, err := http.NewRequestWithContext(t.ctx, "PUT", fmt.Sprintf("http://%s/_template/%s", server.String(), name), templateReader)
	if err != nil {
		return err
	}

	httpRequest.Header.Add("Content-Type", "application/json")
	httpRequest.Header.Add("Content-Length", fmt.Sprintf("%d", templateLen))

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return err
	}
	defer func() {
		bufio.NewReader(httpResponse.Body).WriteTo(ioutil.Discard)
		httpResponse.Body.Close()
	}()
	if httpResponse.StatusCode != 200 {
		body, _ := ioutil.ReadAll(httpResponse.Body)
		return fmt.Errorf("Unexpected status: %s [Body: %s]", httpResponse.Status, body)
	}

	log.Info("[%s] Successfully installed Elasticsearch index template: %s", t.pool.Server(), name)

	return nil
}

// checkTemplate checks if template is already installed at the given server
func (t *transportES) checkTemplate(server *net.TCPAddr, name string) (bool, error) {
	httpRequest, err := http.NewRequestWithContext(t.ctx, "HEAD", fmt.Sprintf("http://%s/_template/%s", server.String(), name), nil)
	if err != nil {
		return false, err
	}

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return false, err
	}
	defer func() {
		bufio.NewReader(httpResponse.Body).WriteTo(ioutil.Discard)
		httpResponse.Body.Close()
	}()
	if httpResponse.StatusCode == 200 {
		return true, nil
	}
	if httpResponse.StatusCode != 404 {
		body, _ := ioutil.ReadAll(httpResponse.Body)
		return false, fmt.Errorf("Unexpected status: %s [Body: %s]", httpResponse.Status, body)
	}

	return false, nil
}

// httpRoutine performs bulk requests to ES
func (t *transportES) httpRoutine(id int) {
	defer func() {
		t.wait.Done()
	}()

	backoffName := fmt.Sprintf("%s:%d Retry", t.pool.Server(), id)
	backoff := core.NewExpBackoff(backoffName, t.config.Retry, t.config.RetryMax)

	for {
		select {
		case <-t.ctx.Done():
			// Forced failure
			return
		case payload := <-t.payloadChan:
			if payload == nil {
				// Graceful shutdown
				log.Info("[%s:%d] Elasticsearch routine stopped gracefully", t.pool.Server(), id)
				return
			}

			lastAckSequence := uint32(0)
			request := newBulkRequest(t.config.IndexPattern, payload.Events())

			for {
				if err := t.performBulkRequest(id, request); err != nil {
					log.Error("[%s:%d] Elasticsearch request failed: %s", t.pool.Server(), id, err)
				}

				if request.AckSequence() != lastAckSequence {
					lastAckSequence = request.AckSequence()

					select {
					case <-t.ctx.Done():
						// Forced failure
						return
					case t.eventChan <- transports.NewAckEvent(t.ctx, payload.Nonce, lastAckSequence):
					}
				}

				if request.Remaining() == 0 {
					break
				}

				if t.retryWait(backoff) {
					break
				}
			}
		}
	}
}

// performBulkRequest performs a bulk request to the Elasticsearch server
func (t *transportES) performBulkRequest(id int, request *bulkRequest) error {
	// Pool Next() is not race-safe
	t.poolMutex.Lock()
	server, err := t.pool.Next()
	t.poolMutex.Unlock()
	if err != nil {
		return err
	}

	// Store what's already created so we can calculate what this specific request created
	created := request.Created()

	defaultIndex, err := request.DefaultIndex()
	if err != nil {
		return err
	}

	var url string
	if t.maxMajorVersion >= 7 {
		// No longer need _type when >= 7
		url = fmt.Sprintf("http://%s/%s/_bulk", server.String(), defaultIndex)
	} else {
		url = fmt.Sprintf("http://%s/%s/_doc/_bulk", server.String(), defaultIndex)
	}
	log.Debug("[%s:%d] Performing Elasticsearch bulk request of %d events via %s", t.pool.Server(), id, request.Remaining(), url)

	request.Reset()
	bodyBuffer := new(bytes.Buffer)
	zlibWriter := gzip.NewWriter(bodyBuffer)
	if _, err := io.Copy(zlibWriter, request); err != nil {
		return err
	}
	if err := zlibWriter.Close(); err != nil {
		return err
	}

	httpRequest, err := http.NewRequestWithContext(t.ctx, "POST", url, bodyBuffer)
	if err != nil {
		return err
	}

	httpRequest.Header.Add("Content-Length", fmt.Sprintf("%d", bodyBuffer.Len()))
	httpRequest.Header.Add("Content-Type", "application/x-ndjson")
	httpRequest.Header.Add("Content-Encoding", "gzip")

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return err
	}
	body, _ := ioutil.ReadAll(httpResponse.Body)
	httpResponse.Body.Close()
	if httpResponse.StatusCode != 200 {
		return fmt.Errorf("Unexpected status: %s [Body: %s]", httpResponse.Status, body)
	}

	response, err := newBulkResponse(body, request)
	if err != nil {
		return fmt.Errorf("Response failed to parse: %s [Body: %s]", err, body)
	}

	if len(response.Errors) != 0 {
		for _, errorValue := range response.Errors {
			log.Warning("[%s:%d] Failed to index event: %s", t.pool.Server(), id, errorValue.Error())
		}
	}

	if request.Remaining() == 0 {
		log.Debug("[%s:%d] Elasticsearch request complete (took %d; created %d; errors %d)", t.pool.Server(), id, response.Took, request.Created()-created, len(response.Errors))
	} else {
		log.Warning("[%s:%d] Elasticsearch request partially complete (took %d; created %d; errors %d; retrying %d)", t.pool.Server(), id, response.Took, request.Created()-created, len(response.Errors), request.Remaining())
	}
	return nil
}

// retryWait waits the backoff timeout before attempting to retry
// It also monitors for shutdown whilst waiting
func (t *transportES) retryWait(backoff *core.ExpBackoff) bool {
	now := time.Now()
	reconnectDue := now.Add(backoff.Trigger())

	select {
	case <-t.ctx.Done():
		// Shutdown request
		return true
	case <-time.After(reconnectDue.Sub(now)):
	}

	return false
}

// Write a message to the transport - only valid after Started transport event received
func (t *transportES) Write(payload *payload.Payload) error {
	// Are we ready?
	t.payloadMutex.Lock()
	defer t.payloadMutex.Unlock()
	if t.payloadChan == nil {
		return errors.New("Invalid connection state")
	}
	t.payloadChan <- payload
	return nil
}

// Ping the remote server - not implemented for HTTP since we close connections after each send
func (t *transportES) Ping() error {
	return nil
}

// Fail the transport
func (t *transportES) Fail() {
	t.shutdownFunc()
}

// Shutdown the transport - only valid after Started transport event received
func (t *transportES) Shutdown() {
	t.payloadMutex.Lock()
	defer t.payloadMutex.Unlock()
	// Trigger graceful shutdown
	if t.payloadChan != nil {
		close(t.payloadChan)
	}
}
