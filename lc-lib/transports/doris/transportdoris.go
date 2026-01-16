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

package doris

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports"
	_ "github.com/go-sql-driver/mysql"
)

var (
	// ErrInvalidState occurs when a send cannot happen because the connection has closed
	ErrInvalidState = errors.New("invalid connection state")

	tableSchemaLock sync.Mutex
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

// transportDoris implements a transport that sends over the Doris HTTP stream load protocol
type transportDoris struct {
	// Constructor
	ctx          context.Context
	shutdownFunc context.CancelFunc
	config       *TransportDorisFactory
	netConfig    *transports.Config
	poolEntry    *addresspool.PoolEntry
	clientCache  map[string]*clientCacheItem
	eventChan    chan<- transports.Event

	// Internal
	payloadChan  chan *payload
	payloadMutex sync.Mutex
	poolMutex    sync.Mutex
	wait         sync.WaitGroup
	tablePattern event.Pattern
	tableMgr     map[string]*tableManager
}

// Factory returns the associated factory
func (t *transportDoris) Factory() transports.TransportFactory {
	return t.config
}

// startController starts the controller
func (t *transportDoris) startController() {
	go t.controllerRoutine()
}

// controllerRoutine is the master routine which handles submission
func (t *transportDoris) controllerRoutine() {
	defer func() {
		// Wait for all routines to close and close all connections
		t.wait.Wait()
		for _, cacheItem := range t.clientCache {
			cacheItem.client.CloseIdleConnections()
		}
		t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Finished, nil)
	}()

	// Create single table manager instance
	t.tableMgr = make(map[string]*tableManager)

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

// prepareTableSchema prepares the table schema by connecting to metadata servers
// and creating or validating the table
func (t *transportDoris) prepareTableSchema(id int, table string) (map[string]string, bool) {
	if tableMgr, ok := t.tableMgr[table]; ok {
		return tableMgr.ColumnDefs(), false
	}
	tableMgr := newTableManager(t.config, table)
	t.tableMgr[table] = tableMgr

	defer tableSchemaLock.Unlock()
	tableSchemaLock.Lock()

	backoffName := fmt.Sprintf("[T %s] Setup Retry", t.poolEntry.Server)
	backoff := core.NewExpBackoff(backoffName, t.config.Retry, t.config.RetryMax)

	metadataEntries, err := addresspool.GeneratePool(t.config.MetadataServers, t.netConfig.Rfc2782Srv, t.netConfig.Rfc2782Service, time.Second*60)
	if err != nil {
		log.Errorf("[T %s] Metadata server lookup failure: %s", t.poolEntry.Server, err)
		return nil, true
	}

MetadataConnectLoop:
	for {
		for _, metadataEntry := range metadataEntries {
			addr, err := metadataEntry.Next()
			if err != nil {
				log.Errorf("[T %s] Metadata server lookup failure: %s", t.poolEntry.Server, err)
				return nil, true
			}

			connected, err := tableMgr.InitializeSchema(t.poolEntry, addr)
			if err == nil {
				// Success
				break MetadataConnectLoop
			}

			// Check if connection failed (retryable) or schema error (fatal)
			if connected {
				// Connected but schema operation failed - fatal error
				log.Errorf("[T %s]{%d}{%s} Failed to initialize Doris table schema: %s", t.poolEntry.Server, id, addr.Desc(), err)
				return nil, true
			}

			// Connection error - try next server
			log.Warningf("[T %s]{%d}{%s} Failed to connect: %s, trying next metadata server", t.poolEntry.Server, id, addr.Desc(), err)
		}

		// All metadata servers failed - wait and retry
		if t.retryWait(backoff) {
			// Shutdown requested during retry
			return nil, true
		}
	}

	return tableMgr.ColumnDefs(), false
}

// httpRoutine performs stream load requests to Doris
func (t *transportDoris) httpRoutine(id int) {
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
				log.Infof("[T %s]{%d} Doris routine stopped gracefully", t.poolEntry.Server, id)
				return
			}

			eventCount := uint32(len(payload.events))

			// If using a static pattern, simplify the requests we need to send
			// statically and quickly
			// Otherwise if a pattern, we need to calculate multiple stream loads,
			// one per table name
			var requests map[string]*streamLoadRequest
			if t.tablePattern.IsStatic() {
				columnDefs, shutdown := t.prepareTableSchema(id, t.tablePattern.String())
				if shutdown {
					return
				}
				// Use static allocation to avoid heap allocation
				requests = map[string]*streamLoadRequest{
					t.config.TablePattern: newStreamLoadRequest(columnDefs, t.config.RestJSONColumn, payload.events),
				}
			} else {
				var eventsByTable map[string][]*event.Event
				if t.tablePattern.IsStatic() {
					eventsByTable = map[string][]*event.Event{
						t.config.TablePattern: payload.events,
					}
				} else {
					eventsByTable = make(map[string][]*event.Event)
					for _, ev := range payload.events {
						tableName, err := t.tablePattern.Format(ev)
						if err != nil {
							log.Errorf("[T %s]{%d} Failed to determine table name for event: %s", t.poolEntry.Server, id, err)
							continue
						}
						eventsByTable[tableName] = append(eventsByTable[tableName], ev)
					}
				}
				requests = make(map[string]*streamLoadRequest)
				for tableName, events := range eventsByTable {
					columnDefs, shutdown := t.prepareTableSchema(id, tableName)
					if shutdown {
						return
					}
					requests[tableName] = newStreamLoadRequest(columnDefs, t.config.RestJSONColumn, events)
				}
			}

			for tableName, request := range requests {
				// Retry until successful or shutdown
				for {
					t.poolMutex.Lock()
					addr, err := t.poolEntry.Next()
					t.poolMutex.Unlock()
					if err == nil {
						err = t.performStreamLoad(addr, id, tableName, request, payload.nonce)
					}
					if err == nil {
						select {
						case <-t.ctx.Done():
							// Forced failure
							return
						case t.eventChan <- transports.NewAckEvent(t.ctx, payload.nonce, eventCount):
						}
						break
					}
					log.Errorf("[T %s]{%d} Doris stream load failed: %s", addr.Desc(), id, err)

					if t.retryWait(backoff) {
						return
					}
				}
			}
		}
	}
}

// performStreamLoad performs a stream load request to the Doris server
func (t *transportDoris) performStreamLoad(addr *addresspool.Address, id int, tableName string, request *streamLoadRequest, nonce *string) error {
	url := fmt.Sprintf("/api/%s/%s/_stream_load", t.config.Database, tableName)
	eventCount := request.EventCount()
	log.Debugf("[T %s]{%d} Performing Doris stream load of %d events to %s", addr.Desc(), id, eventCount, url)

	request.Reset()

	httpRequest, err := t.createRequest(t.ctx, "PUT", addr, url, request)
	if err != nil {
		return err
	}

	// Add any custom load properties
	for key, value := range t.config.LoadProperties {
		httpRequest.Header.Add(key, value)
	}

	httpRequest.Header.Add("Content-Length", fmt.Sprintf("%d", request.Len()))
	httpRequest.Header.Add("Content-Type", "application/json")
	httpRequest.Header.Add("Content-Encoding", "gzip")
	httpRequest.Header.Add("Expect", "100-continue")
	httpRequest.Header.Add("format", "json")
	httpRequest.Header.Add("read_json_by_line", "true")
	httpRequest.Header.Add("label", fmt.Sprintf("log-courier-%s-%x", tableName, *nonce))

	httpResponse, err := t.getClient(addr).Do(httpRequest)
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(httpResponse.Body)
	httpResponse.Body.Close()

	if httpResponse.StatusCode != 200 {
		return fmt.Errorf("unexpected status: %s [Body: %s]", httpResponse.Status, body)
	}

	response, err := newStreamLoadResponse(body)
	if err != nil {
		return fmt.Errorf("response failed to parse: %s [Body: %s]", err, body)
	}

	if response.Status != "Success" && response.Status != "Publish Timeout" {
		return fmt.Errorf("stream load failed with status: %s [Message: %s] [Comment: %s] [FirstErrorMessage: %s] [ErrorURL: %s]", response.Status, response.Message, response.Comment, response.FirstErrorMessage, response.ErrorURL)
	}

	log.Debugf("[T %s]{%d} Doris stream load complete (txnid: %d; label: %s; loaded %d; filtered %d; time %dms)", addr.Desc(), id, response.TxnID, response.Label, response.NumberLoadedRows, response.NumberFilteredRows, response.LoadTimeMs)

	return nil
}

// retryWait waits the backoff timeout before attempting to retry
// It also monitors for shutdown whilst waiting
func (t *transportDoris) retryWait(backoff *core.ExpBackoff) bool {
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
func (t *transportDoris) SendEvents(nonce string, events []*event.Event) error {
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
func (t *transportDoris) Ping() error {
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
func (t *transportDoris) Fail() {
	t.shutdownFunc()
}

// Shutdown the transport - only valid after Started transport event received
func (t *transportDoris) Shutdown() {
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
func (t *transportDoris) createRequest(ctx context.Context, method string, addr *addresspool.Address, url string, body io.Reader) (*http.Request, error) {
	var scheme string
	if t.config.transport == TransportDorisHTTPS {
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
func (t *transportDoris) getClient(addr *addresspool.Address) *http.Client {
	t.poolMutex.Lock()
	defer t.poolMutex.Unlock()

	now := time.Now()
	expires := time.Now().Add(time.Second * 300)
	cacheItem, ok := t.clientCache[addr.Addr().String()]
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

	t.clientCache[addr.Addr().String()] = &clientCacheItem{client, expires}
	return client
}
