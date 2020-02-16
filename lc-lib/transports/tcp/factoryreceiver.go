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

package tcp

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// ReceiverTCPFactory holds the configuration from the configuration file
// It allows creation of ReceiverTCP instances that use this configuration
type ReceiverTCPFactory struct {
	transport string

	config *config.Config

	SSLCertificate string   `config:"ssl certificate"`
	SSLKey         string   `config:"ssl key"`
	SSLClientCA    []string `config:"ssl client ca"`

	hostportRegexp  *regexp.Regexp
	certificate     *tls.Certificate
	certificateList []*x509.Certificate
	caList          []*x509.Certificate
}

// NewReceiverTCPFactory create a new ReceiverTCPFactory from the provided
// configuration data, reporting back any configuration errors it discovers.
func NewReceiverTCPFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (transports.ReceiverFactory, error) {
	var err error

	ret := &ReceiverTCPFactory{
		config:         p.Config(),
		transport:      name,
		hostportRegexp: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
	}

	if err = p.Populate(ret, unUsed, configPath, true); err != nil {
		return nil, err
	}

	// Only allow SSL configurations if using TLS
	if name == TransportTCPTLS {
		if len(ret.SSLCertificate) > 0 || len(ret.SSLKey) > 0 {
			if len(ret.SSLCertificate) == 0 {
				return nil, errors.New("ssl key is only valid with a matching ssl certificate")
			}

			if len(ret.SSLKey) == 0 {
				return nil, errors.New("ssl key must be specified when a ssl certificate is provided")
			}

			certificate, err := tls.LoadX509KeyPair(ret.SSLCertificate, ret.SSLKey)
			if err != nil {
				return nil, fmt.Errorf("failed loading client ssl certificate: %s", err)
			}

			ret.certificate = &certificate

			for _, certBytes := range ret.certificate.Certificate {
				thisCert, err := x509.ParseCertificate(certBytes)
				if err != nil {
					return nil, fmt.Errorf("failed loading client ssl certificate: %s", err)
				}
				ret.certificateList = append(ret.certificateList, thisCert)
			}
		}

		for _, clientCA := range ret.SSLClientCA {
			pemdata, err := ioutil.ReadFile(clientCA)
			if err != nil {
				return nil, fmt.Errorf("failure reading CA certificate %s: %s\n", clientCA, err)
			}
			rest := pemdata
			var block *pem.Block
			var pemBlockNum = 1
			for {
				block, rest = pem.Decode(rest)
				if block != nil {
					if block.Type != "CERTIFICATE" {
						return nil, fmt.Errorf("block %d of %s does not contain a certificate\n", pemBlockNum, clientCA)
					}
					cert, err := x509.ParseCertificate(block.Bytes)
					if err != nil {
						return nil, fmt.Errorf("failed to parse CA certificate in block %d of %s\n", pemBlockNum, clientCA)
					}
					ret.caList = append(ret.caList, cert)
					pemBlockNum++
				} else {
					break
				}
			}
		}
	} else {
		if len(ret.SSLCertificate) > 0 || len(ret.SSLKey) > 0 || len(ret.SSLClientCA) > 0 {
			return nil, fmt.Errorf("transport tcp does not support ssl certificate, ssl key or ssl client ca configurations")
		}
	}

	return ret, nil
}

// NewReceiver returns a new Receiver interface using the settings from the
// ReceiverTCPFactory.
func (f *ReceiverTCPFactory) NewReceiver(context interface{}, pool *addresspool.Pool, eventChan chan<- transports.Event) transports.Receiver {
	ret := &ReceiverTCP{
		config:         f,
		netConfig:      transports.FetchReceiverConfig(f.config),
		context:        context,
		pool:           pool,
		eventChan:      eventChan,
		controllerChan: make(chan error),
		connectionChan: make(chan *socketMessage),
	}

	go ret.controller()

	return ret
}

// Register the transports
func init() {
	transports.RegisterReceiver(TransportTCPTCP, NewReceiverTCPFactory)
	transports.RegisterReceiver(TransportTCPTLS, NewReceiverTCPFactory)
}
