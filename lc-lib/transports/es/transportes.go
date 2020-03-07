/*
 * Copyright 2014-2015 Jason Woods.
 *
 * This file is a modification of code from Logstash Forwarder.
 * Copyright 2012-2013 Jordan Sissel and contributors.
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
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/payload"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// transportES implements a transport that sends over the ES HTTP protocol
type transportES struct {
	// Constructor
	config          *TransportESFactory
	netConfig       *transports.Config
	finishOnFail    bool
	context         interface{}
	pool            *addresspool.Pool
	eventChan       chan<- transports.Event
	shutdownContext context.Context
	shutdownFunc    context.CancelFunc

	// Internal
	// payloadMutex is so we can easily discard existing sendChan and its contents each time we reset
	payloadChan  chan *payload.Payload
	payloadMutex sync.Mutex
	poolMutex    sync.Mutex
	wait         sync.WaitGroup
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
		t.eventChan <- transports.NewStatusEvent(t.context, transports.Finished)
	}()

	// Setup payload chan with max write count of pending payloads
	t.payloadMutex.Lock()
	t.payloadChan = make(chan *payload.Payload, t.netConfig.MaxPendingPayloads)
	t.payloadMutex.Unlock()

	// TODO: Create index mapping template

	t.eventChan <- transports.NewStatusEvent(t.context, transports.Started)

	// Start secondary http routines
	for i := 1; i < t.config.Routines; i++ {
		t.wait.Add(1)
		go t.httpRoutine(i)
	}

	// Become the main http routine
	t.wait.Add(1)
	t.httpRoutine(0)
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
		case <-t.shutdownContext.Done():
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
					case <-t.shutdownContext.Done():
						// Forced failure
						return
					case t.eventChan <- transports.NewAckEvent(t.context, payload.Nonce, lastAckSequence):
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

// retryWait waits the backoff timeout before attempting to retry
// It also monitors for shutdown whilst waiting
func (t *transportES) retryWait(backoff *core.ExpBackoff) bool {
	now := time.Now()
	reconnectDue := now.Add(backoff.Trigger())

	select {
	case <-t.shutdownContext.Done():
		// Shutdown request
		return true
	case <-time.After(reconnectDue.Sub(now)):
	}

	return false
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

	url := fmt.Sprintf("http://%s/%s/_bulk", server.String(), defaultIndex)
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

	httpRequest, err := http.NewRequestWithContext(
		t.shutdownContext,
		"POST",
		url,
		bodyBuffer,
	)
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
	if httpResponse.StatusCode != 200 {
		readall, _ := ioutil.ReadAll(httpResponse.Body)
		log.Warning("%s", readall)
		return errors.New(httpResponse.Status)
	}

	response, err := newBulkResponse(httpResponse.Body, request)
	if err != nil {
		return fmt.Errorf("Response failed to parse: %s", err)
	}

	if request.Remaining() == 0 {
		log.Debug("[%s:%d] Elasticsearch request successful (took %d; created %d)", t.pool.Server(), id, response.Took, request.Created()-created)
	} else {
		log.Warning("[%s:%d] Elasticsearch request partially successful (took %d, created %d)", t.pool.Server(), id, response.Took, request.Created()-created)
	}
	return nil
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
