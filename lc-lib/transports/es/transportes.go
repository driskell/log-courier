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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports"
)

var (
	// ErrInvalidState occurs when a send cannot happen because the connection has closed
	ErrInvalidState = errors.New("invalid connection state")
)

// payload contains nonce and events information
type payload struct {
	nonce  *string
	events []*event.Event
}

type clientCacheItem struct {
	client  *http.Client
	expires time.Time
}

// transportES implements a transport that sends over the ES HTTP protocol
type transportES struct {
	// Constructor
	ctx          context.Context
	shutdownFunc context.CancelFunc
	config       *TransportESFactory
	netConfig    *transports.Config
	poolEntry    *addresspool.PoolEntry
	clientCache  map[string]*clientCacheItem
	eventChan    chan<- transports.Event

	// Internal
	// payloadMutex is so we can easily discard existing sendChan and its contents each time we reset
	payloadChan     chan *payload
	payloadMutex    sync.Mutex
	poolMutex       sync.Mutex
	wait            sync.WaitGroup
	nodeInfo        *nodeInfo
	maxMajorVersion int
}

// Factory returns the associated factory
func (t *transportES) Factory() transports.TransportFactory {
	return t.config
}

// startController starts the controller
func (t *transportES) startController() {
	go t.controllerRoutine()
}

// controllerRoutine is the master routine which handles submission
func (t *transportES) controllerRoutine() {
	defer func() {
		// Wait for all routines to close and close all connections
		t.wait.Wait()
		for _, cacheItem := range t.clientCache {
			cacheItem.client.CloseIdleConnections()
		}
		t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Finished, nil)
	}()

	if t.setupAssociation() {
		// Shutdown was requested
		return
	}

	// Setup payload chan with max write count of pending payloads
	t.payloadMutex.Lock()
	t.payloadChan = make(chan *payload, t.netConfig.MaxPendingPayloads)
	t.payloadMutex.Unlock()

	t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Started, nil)

	// Start secondary http routines
	for i := 1; i < t.config.Routines; i++ {
		t.wait.Add(1)
		go t.httpRoutine(i)
	}

	// Become the main http routine
	t.wait.Add(1)
	t.httpRoutine(0)

	// Ensure all resources for the cancel are cleaned up
	t.shutdownFunc()
}

// setupAssociation gathers cluster information and installs templates
func (t *transportES) setupAssociation() bool {
	backoffName := fmt.Sprintf("[T %s] Setup Retry", t.poolEntry.Server)
	backoff := core.NewExpBackoff(backoffName, t.config.Retry, t.config.RetryMax)

	for {
		addr, err := t.poolEntry.Next()
		if err != nil {
			log.Errorf("[T %s] Failed to resolve Elasticsearch node address: %s", addr.Desc(), err)
		} else if err := t.populateNodeInfo(addr); err != nil {
			log.Errorf("[T %s] Failed to fetch Elasticsearch node information: %s", addr.Desc(), err)
		} else if err := t.installTemplate(addr); err != nil {
			log.Errorf("[T %s] Failed to install Elasticsearch index template: %s", addr.Desc(), err)
		} else {
			return false
		}

		if t.retryWait(backoff) {
			break
		}
	}

	// Shutdown
	return true
}

// populateNodeInfo populates the nodeInfo structure
func (t *transportES) populateNodeInfo(addr *addresspool.Address) error {
	httpRequest, err := t.createRequest(t.ctx, "GET", addr, "/_nodes/http", nil)
	if err != nil {
		return err
	}

	httpResponse, err := t.getClient(addr).Do(httpRequest)
	if err != nil {
		return err
	}
	defer func() {
		bufio.NewReader(httpResponse.Body).WriteTo(io.Discard)
		httpResponse.Body.Close()
	}()
	if httpResponse.StatusCode != 200 {
		body, _ := io.ReadAll(httpResponse.Body)
		return fmt.Errorf("unexpected status: %s [Body: %s]", httpResponse.Status, body)
	}

	decoder := json.NewDecoder(httpResponse.Body)
	if err := decoder.Decode(&t.nodeInfo); err != nil {
		return err
	}

	t.maxMajorVersion, err = t.nodeInfo.MaxMajorVersion()
	if err != nil {
		return fmt.Errorf("failed to calculate maximum version number for cluster: %s", err)
	}

	log.Infof("[T %s] Successfully retrieved Elasticsearch node information (major version: %d)", addr.Desc(), t.maxMajorVersion)

	return nil
}

// installTemplate checks if template installation is needed
func (t *transportES) installTemplate(addr *addresspool.Address) error {
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
		case 7:
			template = esTemplate7
		case 6:
			template = esTemplate6
		case 5:
			template = esTemplate5
			if len(t.config.TemplatePatterns) > 1 {
				return fmt.Errorf("the Elasticsearch major version %d does not support multiple template patterns", t.maxMajorVersion)
			}
			template = strings.ReplaceAll(template, "$INDEXPATTERNSINGLE$", t.config.templatePatternSingleJSON)
		default:
			return fmt.Errorf("the Elasticsearch major version %d is unsupported", t.maxMajorVersion)
		}
		template = strings.ReplaceAll(template, "$INDEXPATTERNS$", t.config.templatePatternsJSON)
		templateReader = strings.NewReader(template)
		templateLen = len(template)
	}

	installed, err := t.checkTemplate(addr, name)
	if err != nil {
		return err
	} else if installed {
		return nil
	}

	httpRequest, err := t.createRequest(t.ctx, "PUT", addr, fmt.Sprintf("/_template/%s", name), templateReader)
	if err != nil {
		return err
	}

	httpRequest.Header.Add("Content-Type", "application/json")
	httpRequest.Header.Add("Content-Length", fmt.Sprintf("%d", templateLen))

	httpResponse, err := t.getClient(addr).Do(httpRequest)
	if err != nil {
		return err
	}
	defer func() {
		bufio.NewReader(httpResponse.Body).WriteTo(io.Discard)
		httpResponse.Body.Close()
	}()
	if httpResponse.StatusCode != 200 {
		body, _ := io.ReadAll(httpResponse.Body)
		return fmt.Errorf("unexpected status: %s [Body: %s]", httpResponse.Status, body)
	}

	log.Infof("[T %s] Successfully installed Elasticsearch index template: %s", addr.Desc(), name)

	return nil
}

// checkTemplate checks if template is already installed at the given server
func (t *transportES) checkTemplate(addr *addresspool.Address, name string) (bool, error) {
	httpRequest, err := t.createRequest(t.ctx, "HEAD", addr, fmt.Sprintf("/_template/%s", name), nil)
	if err != nil {
		return false, err
	}

	httpResponse, err := t.getClient(addr).Do(httpRequest)
	if err != nil {
		return false, err
	}
	defer func() {
		bufio.NewReader(httpResponse.Body).WriteTo(io.Discard)
		httpResponse.Body.Close()
	}()
	if httpResponse.StatusCode == 200 {
		return true, nil
	}
	if httpResponse.StatusCode != 404 {
		body, _ := io.ReadAll(httpResponse.Body)
		return false, fmt.Errorf("unexpected status: %s [Body: %s]", httpResponse.Status, body)
	}

	return false, nil
}

// httpRoutine performs bulk requests to ES
func (t *transportES) httpRoutine(id int) {
	defer func() {
		t.wait.Done()
	}()

	backoffName := fmt.Sprintf("%s:%d Retry", t.poolEntry.Server, id)
	backoff := core.NewExpBackoff(backoffName, t.config.Retry, t.config.RetryMax)

	for {
		select {
		case <-t.ctx.Done():
			// Forced failure
			return
		case payload := <-t.payloadChan:
			if payload == nil {
				// Graceful shutdown
				log.Infof("[T %s]{%d} Elasticsearch routine stopped gracefully", t.poolEntry.Server, id)
				return
			}

			lastAckSequence := uint32(0)
			request := newBulkRequest(t.config.IndexPattern, payload.events)

			for {
				// Pool Next() is not race-safe
				t.poolMutex.Lock()
				addr, err := t.poolEntry.Next()
				t.poolMutex.Unlock()
				if err == nil {
					err = t.performBulkRequest(addr, id, request)
				}
				if err != nil {
					log.Errorf("[T %s]{%d} Elasticsearch request failed: %s", addr.Desc(), id, err)
				}

				if request.AckSequence() != lastAckSequence {
					lastAckSequence = request.AckSequence()

					select {
					case <-t.ctx.Done():
						// Forced failure
						return
					case t.eventChan <- transports.NewAckEvent(t.ctx, payload.nonce, lastAckSequence):
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
func (t *transportES) performBulkRequest(addr *addresspool.Address, id int, request *bulkRequest) error {
	// Store what's already created so we can calculate what this specific request created
	created := request.Created()

	defaultIndex, err := request.DefaultIndex()
	if err != nil {
		return err
	}

	var url string
	if t.maxMajorVersion >= 7 {
		// No longer need _type when >= 7
		url = fmt.Sprintf("/%s/_bulk", defaultIndex)
	} else {
		url = fmt.Sprintf("/%s/_doc/_bulk", defaultIndex)
	}
	log.Debugf("[T %s]{%d} Performing Elasticsearch bulk request of %d events to %s", addr.Desc(), id, request.Remaining(), url)

	request.Reset()
	bodyBuffer := new(bytes.Buffer)
	zlibWriter := gzip.NewWriter(bodyBuffer)
	if _, err := io.Copy(zlibWriter, request); err != nil {
		return err
	}
	if err := zlibWriter.Close(); err != nil {
		return err
	}

	httpRequest, err := t.createRequest(t.ctx, "POST", addr, url, bodyBuffer)
	if err != nil {
		return err
	}

	httpRequest.Header.Add("Content-Length", fmt.Sprintf("%d", bodyBuffer.Len()))
	httpRequest.Header.Add("Content-Type", "application/x-ndjson")
	httpRequest.Header.Add("Content-Encoding", "gzip")

	httpResponse, err := t.getClient(addr).Do(httpRequest)
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(httpResponse.Body)
	httpResponse.Body.Close()
	if httpResponse.StatusCode != 200 {
		return fmt.Errorf("unexpected status: %s [Body: %s]", httpResponse.Status, body)
	}

	response, err := newBulkResponse(body, request)
	if err != nil {
		return fmt.Errorf("response failed to parse: %s [Body: %s]", err, body)
	}

	clusterBlocked := 0
	var lastError *bulkResponseError
	lastCount := 0
	if len(response.Errors) != 0 {
		for _, errorValue := range response.Errors {
			if errorValue.Type == "cluster_block_exception" {
				clusterBlocked += 1
			}
			if lastError == nil {
				lastError = errorValue
				lastCount = 1
			} else if errorValue.SameAs(lastError) {
				lastCount += 1
			} else {
				logEventErrors(addr.Desc(), id, lastError, lastCount)
				lastError = errorValue
				lastCount = 1
			}
		}
	}
	if lastError != nil {
		logEventErrors(addr.Desc(), id, lastError, lastCount)
	}

	// If cluster is blocked on all messages, allow a retry to occur by throwing an error
	if len(response.Errors) > 0 && clusterBlocked == len(response.Errors) {
		return fmt.Errorf("cluster is blocked")
	}

	if request.Remaining() == 0 {
		log.Debugf("[T %s]{%d} Elasticsearch request complete (took %dms; created %d; errors %d)", addr.Desc(), id, response.Took, request.Created()-created, len(response.Errors))
	} else {
		log.Warningf("[T %s]{%d} Elasticsearch request partially complete (took %dms; created %d; errors %d; retrying %d)", addr.Desc(), id, response.Took, request.Created()-created, len(response.Errors), request.Remaining())
	}
	return nil
}

// logEventErrors logs errors from events
func logEventErrors(desc string, id int, lastError *bulkResponseError, lastCount int) {
	if lastCount > 1 {
		log.Warningf("[T %s]{%d} Failed to index event: %s [Repeated for %d more events]", desc, id, lastError.Error(), lastCount)
	} else {
		log.Warningf("[T %s]{%d} Failed to index event: %s", desc, id, lastError.Error())
	}
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

// SendEvents sends events to the transport - only valid after Started transport event received
func (t *transportES) SendEvents(nonce string, events []*event.Event) error {
	// Are we ready?
	t.payloadMutex.Lock()
	defer t.payloadMutex.Unlock()
	if t.payloadChan == nil {
		return ErrInvalidState
	}
	t.payloadChan <- &payload{&nonce, events}
	return nil
}

// Ping the remote server - not implemented for HTTP since we close connections after each send
// Immediately respond with a pong
func (t *transportES) Ping() error {
	go func() {
		log.Debugf("[T %s] Responding with pong", t.poolEntry.Server)
		select {
		case <-t.ctx.Done():
			// Forced failure
			return
		case t.eventChan <- transports.NewPongEvent(t.ctx):
		}
	}()
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
	if t.payloadChan == nil {
		// No connection active so just fail
		t.shutdownFunc()
	} else {
		// Trigger graceful shutdown
		close(t.payloadChan)
	}
}

// createRequest creates a new http.Request and adds default headers
func (t *transportES) createRequest(ctx context.Context, method string, addr *addresspool.Address, url string, body io.Reader) (*http.Request, error) {
	var scheme string
	if t.config.transport == TransportESHTTPS {
		scheme = "https"
	} else {
		scheme = "http"
	}

	request, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("%s://%s%s", scheme, addr.Addr().String(), url), body)
	if err != nil {
		return nil, err
	}

	if t.config.Username != "" && t.config.Password != "" {
		request.SetBasicAuth(t.config.Username, t.config.Password)
	}

	return request, nil
}

// getClient returns a http.Client for the given server
func (t *transportES) getClient(addr *addresspool.Address) *http.Client {
	t.poolMutex.Lock()
	defer t.poolMutex.Unlock()

	now := time.Now()
	expires := time.Now().Add(time.Second * 300)
	cacheItem, ok := t.clientCache[addr.Host()]
	if ok {
		cacheItem.expires = expires
		return cacheItem.client
	}

	for key, cacheItem := range t.clientCache {
		if cacheItem.expires.Before(now) {
			cacheItem.client.CloseIdleConnections()
			delete(t.clientCache, key)
		}
	}

	certPool := x509.NewCertPool()
	for _, cert := range t.config.CaList {
		certPool.AddCert(cert)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSHandshakeTimeout: t.netConfig.Timeout,
			TLSClientConfig: &tls.Config{
				RootCAs:    certPool,
				ServerName: addr.Host(),
				MinVersion: t.config.MinTLSVersion,
				MaxVersion: t.config.MaxTLSVersion,
			},
		},
		Timeout: t.netConfig.Timeout,
	}

	t.clientCache[addr.Host()] = &clientCacheItem{client, expires}
	return client
}
