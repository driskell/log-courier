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

package tcp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/transports"
)

const (
	defaultSSLVerifyPeers = true
)

// ReceiverTCPFactory holds the configuration from the configuration file
// It allows creation of ReceiverTCP instances that use this configuration
type ReceiverTCPFactory struct {
	// Constructor
	config         *config.Config
	transport      string
	hostportRegexp *regexp.Regexp

	// Configuration
	SSLCertificate string   `config:"ssl certificate"`
	SSLKey         string   `config:"ssl key"`
	SSLClientCA    []string `config:"ssl client ca"`
	SSLVerifyPeers bool     `config:"verify_peers"`
	MinTLSVersion  string   `config:"min tls version"`
	MaxTLSVersion  string   `config:"max tls version"`

	// Internal
	certificate     *tls.Certificate
	certificateList []*x509.Certificate
	caList          []*x509.Certificate
	minTLSVersion   uint16
	maxTLSVersion   uint16
}

// NewReceiverTCPFactory create a new ReceiverTCPFactory from the provided
// configuration data, reporting back any configuration errors it discovers.
func NewReceiverTCPFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (transports.ReceiverFactory, error) {
	ret := &ReceiverTCPFactory{
		config:         p.Config(),
		transport:      name,
		hostportRegexp: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
		SSLVerifyPeers: defaultSSLVerifyPeers,
	}
	if err := p.Populate(ret, unUsed, configPath, true); err != nil {
		return nil, err
	}
	return ret, nil
}

// Validate the configuration
func (f *ReceiverTCPFactory) Validate(p *config.Parser, configPath string) (err error) {
	// Check tls versions
	f.minTLSVersion, err = parseTLSVersion(f.MinTLSVersion, defaultMinTLSVersion)
	if err != nil {
		return err
	}
	f.maxTLSVersion, err = parseTLSVersion(f.MaxTLSVersion, defaultMaxTLSVersion)
	if err != nil {
		return err
	}

	// Only allow SSL configurations if using TLS
	if f.transport != TransportTCPTLS {
		if len(f.SSLCertificate) > 0 || len(f.SSLKey) > 0 || len(f.SSLClientCA) > 0 {
			return fmt.Errorf("Transport 'tcp' does not support 'ssl certificate', 'ssl key' or 'ssl client ca' configurations")
		}
		return nil
	}

	if len(f.SSLCertificate) > 0 || len(f.SSLKey) > 0 {
		if len(f.SSLCertificate) == 0 {
			return errors.New("The 'ssl key' option is only valid with a matching 'ssl certificate'")
		}

		if len(f.SSLKey) == 0 {
			return errors.New("The 'ssl key' option must be specified when an 'ssl certificate' is provided")
		}

		certificate, err := tls.LoadX509KeyPair(f.SSLCertificate, f.SSLKey)
		if err != nil {
			return fmt.Errorf("Failed loading 'ssl certificate': %s", err)
		}

		f.certificate = &certificate

		for _, certBytes := range f.certificate.Certificate {
			thisCert, err := x509.ParseCertificate(certBytes)
			if err != nil {
				return fmt.Errorf("Failed loading 'ssl certificate': %s", err)
			}
			f.certificateList = append(f.certificateList, thisCert)
		}
	}

	for _, clientCA := range f.SSLClientCA {
		pemdata, err := ioutil.ReadFile(clientCA)
		if err != nil {
			return fmt.Errorf("Failure reading CA certificate '%s': %s", clientCA, err)
		}
		rest := pemdata
		var block *pem.Block
		var pemBlockNum = 1
		for {
			block, rest = pem.Decode(rest)
			if block != nil {
				if block.Type != "CERTIFICATE" {
					return fmt.Errorf("Failure loading 'tls' receiver 'ssl ca': block %d of '%s' does not contain a certificate", pemBlockNum, clientCA)
				}
				cert, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					return fmt.Errorf("Failure loading 'tls' receiver 'ssl ca': failed to parse CA certificate in block %d of '%s'", pemBlockNum, clientCA)
				}
				f.caList = append(f.caList, cert)
				pemBlockNum++
			} else {
				break
			}
		}
	}

	return nil
}

// NewReceiver returns a new Receiver interface using the settings from the ReceiverTCPFactory
func (f *ReceiverTCPFactory) NewReceiver(ctx context.Context, pool *addresspool.Pool, eventChan chan<- transports.Event) transports.Receiver {
	ctx, shutdownFunc := context.WithCancel(ctx)

	ret := &receiverTCP{
		ctx:          ctx,
		shutdownFunc: shutdownFunc,
		config:       f,
		pool:         pool,
		eventChan:    eventChan,
		connections:  make(map[*connection]*connection),
		// TODO: Own values
		backoff: core.NewExpBackoff(pool.Server()+" Receiver Reset", defaultReconnect, defaultReconnectMax),
	}

	ret.startController()
	return ret
}

// Register the transports
func init() {
	transports.RegisterReceiver(TransportTCPTCP, NewReceiverTCPFactory)
	transports.RegisterReceiver(TransportTCPTLS, NewReceiverTCPFactory)
}
