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

package transports

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/driskell/log-courier/lc-lib/config"
)

var (
	// ErrCongestion represents temporary congestion, rather than failure
	ErrCongestion error = errors.New("Congestion")

	// ErrInvalidState occurs when a send cannot happen because the connection has closed
	ErrInvalidState = errors.New("invalid connection state")

	// NilNonce represents the displayed nonce for an event bundle where the source does not use nonces
	NilNonce string = "-"
)

// TransportContext
type TransportContext string

const (
	// ContextConnection provides a value representing an individual connection
	// The returned interface should be treated opaque outside of the relevant transport package
	ContextConnection TransportContext = "connection"

	// ContextReceiver provides the Receiver that a connection relates to
	ContextReceiver TransportContext = "receiver"
)

// StatusChange holds a value that represents a change in transport status
type StatusChange int

// Transport status change signals
const (
	Started StatusChange = iota
	Failed
	Finished
)

// Event is the interface implemented by all event structures
type Event interface {
	Context() context.Context
}

type AckEvent interface {
	Event
	Nonce() *string
	Sequence() uint32
}

type EventsEvent interface {
	Event
	Events() []map[string]interface{}
	Nonce() *string
	Count() uint32
}

// StatusEvent contains information about a status change for a transport
type StatusEvent struct {
	context      context.Context
	statusChange StatusChange
	err          error
}

var _ Event = (*StatusEvent)(nil)

// NewStatusEvent generates a new StatusEvent for the given context
func NewStatusEvent(context context.Context, statusChange StatusChange, err error) *StatusEvent {
	return &StatusEvent{
		context:      context,
		statusChange: statusChange,
		err:          err,
	}
}

// Context returns the endpoint associated with this event
func (e *StatusEvent) Context() context.Context {
	return e.context
}

// StatusChange returns the status change value
func (e *StatusEvent) StatusChange() StatusChange {
	return e.statusChange
}

// StatusChange returns the error associated with a Failed status
func (e *StatusEvent) Err() error {
	return e.err
}

// ConnectEvent marks the start of a new connection on a reciver
type ConnectEvent struct {
	context context.Context
	remote  string
	desc    string
}

var _ Event = (*ConnectEvent)(nil)

// NewConnectEvent generates a new ConnectEvent for the given Endpoint
func NewConnectEvent(context context.Context, remote string, desc string) *ConnectEvent {
	return &ConnectEvent{
		context: context,
		remote:  remote,
		desc:    desc,
	}
}

// Context returns the endpoint associated with this event
func (e *ConnectEvent) Context() context.Context {
	return e.context
}

// Remote returns the identity of the remote side
func (e *ConnectEvent) Remote() string {
	return e.remote
}

// Desc returns a description for the remote
func (e *ConnectEvent) Desc() string {
	return e.desc
}

// EndEvent marks the end of a stream of events from an endpoint
type EndEvent struct {
	context context.Context
}

var _ Event = (*EndEvent)(nil)

// NewEndEvent generates a new EndEvent for the given Endpoint
func NewEndEvent(context context.Context) *EndEvent {
	return &EndEvent{
		context: context,
	}
}

// Context returns the endpoint associated with this event
func (e *EndEvent) Context() context.Context {
	return e.context
}

// PongEvent is received when a transport has responded to a Ping() request
type PongEvent struct {
	context context.Context
}

var _ Event = (*PongEvent)(nil)

// NewPongEvent generates a new PongEvent for the given Endpoint
func NewPongEvent(context context.Context) *PongEvent {
	return &PongEvent{
		context: context,
	}
}

// Context returns the endpoint associated with this event
func (e *PongEvent) Context() context.Context {
	return e.context
}

// PingEvent is received when a transport has responded to a Ping() request
type PingEvent struct {
	context context.Context
}

var _ Event = (*PingEvent)(nil)

// NewPingEvent generates a new PingEvent for the given Endpoint
func NewPingEvent(context context.Context) *PingEvent {
	return &PingEvent{
		context: context,
	}
}

// Context returns the endpoint associated with this event
func (e *PingEvent) Context() context.Context {
	return e.context
}

// ackEvent contains information on which events have been acknowledged
type ackEvent struct {
	context  context.Context
	nonce    *string
	sequence uint32
}

var _ AckEvent = (*ackEvent)(nil)

// NewAckEvent generates a new AckEvent for the given Endpoint
func NewAckEvent(context context.Context, nonce *string, sequence uint32) AckEvent {
	return &ackEvent{
		context:  context,
		nonce:    nonce,
		sequence: sequence,
	}
}

// Context returns the endpoint associated with this event
func (e *ackEvent) Context() context.Context {
	return e.context
}

// Nonce returns the nonce value
func (e *ackEvent) Nonce() *string {
	return e.nonce
}

// Sequence returns the sequence value
func (e *ackEvent) Sequence() uint32 {
	return e.sequence
}

// eventsEvent contains information about an events bundle
type eventsEvent struct {
	context context.Context
	nonce   *string
	events  []map[string]interface{}
}

var _ EventsEvent = (*eventsEvent)(nil)

// NewEventsEvent generates a new EventsEvent for the given bundle of events
func NewEventsEvent(context context.Context, nonce *string, events []map[string]interface{}) EventsEvent {
	return &eventsEvent{
		context: context,
		nonce:   nonce,
		events:  events,
	}
}

// Context returns the endpoint associated with this event
func (e *eventsEvent) Context() context.Context {
	return e.context
}

// Nonce returns the nonce value
func (e *eventsEvent) Nonce() *string {
	return e.nonce
}

// Events returns the events
func (e *eventsEvent) Events() []map[string]interface{} {
	return e.events
}

// Count returns the number of events in the payload
func (e *eventsEvent) Count() uint32 {
	return uint32(len(e.events))
}

// ParseTLSVersion parses a TLS version string into the tls library value for min/max config
// We explicitly refuse SSLv3 to mitigate POODLE vulnerability
func ParseTLSVersion(version string, fallback uint16) (uint16, error) {
	switch version {
	case "":
		return fallback, nil
	case "1.0":
		return tls.VersionTLS10, nil
	case "1.1":
		return tls.VersionTLS11, nil
	case "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	}
	return fallback, fmt.Errorf("invalid or unknown TLS version: '%s'", version)
}

// GetTlsVersionAsString returns a string representation of the TLS version
func GetTlsVersionAsString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLSv1"
	case tls.VersionTLS11:
		return "TLSv1.1"
	case tls.VersionTLS12:
		return "TLSv1.2"
	case tls.VersionTLS13:
		return "TLSv1.3"
	}
	return fmt.Sprintf("Unknown (%d)", version)
}

// AddCertificates returns a new slice containing the given certificate list and the contents of the given file added
func AddCertificates(certificateList []*x509.Certificate, file string) ([]*x509.Certificate, error) {
	pemdata, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	rest := pemdata
	var block *pem.Block
	var pemBlockNum = 1
	for {
		block, rest = pem.Decode(rest)
		if block != nil {
			if block.Type != "CERTIFICATE" {
				return nil, fmt.Errorf("block %d does not contain a certificate", pemBlockNum)
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse CA certificate in block %d", pemBlockNum)
			}
			certificateList = append(certificateList, cert)
			pemBlockNum++
		} else {
			break
		}
	}
	return certificateList, nil
}

// init registers this module provider
func init() {
	config.RegisterAvailable("transports", AvailableTransports)
	config.RegisterAvailable("receivers", AvailableReceivers)
}
