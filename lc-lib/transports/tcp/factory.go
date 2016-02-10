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

package transports

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// TransportTCPFactory holds the configuration from the configuration file
// It allows creation of TransportTCP instances that use this configuration
type TransportTCPFactory struct {
	transport string

	SSLCertificate string `config:"ssl certificate"`
	SSLKey         string `config:"ssl key"`
	SSLCA          string `config:"ssl ca"`

	hostportRegexp *regexp.Regexp
	tlsConfig      tls.Config
	netConfig      *config.Network
}

// NewTransportTCPFactory create a new TransportTCPFactory from the provided
// configuration data, reporting back any configuration errors it discovers.
func NewTransportTCPFactory(config *config.Config, configPath string, unused map[string]interface{}, name string) (interface{}, error) {
	var err error

	ret := &TransportTCPFactory{
		transport:      name,
		hostportRegexp: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
		netConfig:      &config.Network,
	}

	// Only allow SSL configurations if this is "tls"
	if name == "tls" {
		if err = config.PopulateConfig(ret, configPath, unused); err != nil {
			return nil, err
		}

		if len(ret.SSLCertificate) > 0 && len(ret.SSLKey) > 0 {
			cert, err := tls.LoadX509KeyPair(ret.SSLCertificate, ret.SSLKey)
			if err != nil {
				return nil, fmt.Errorf("Failed loading client ssl certificate: %s", err)
			}

			ret.tlsConfig.Certificates = []tls.Certificate{cert}
		}

		if len(ret.SSLCA) > 0 {
			ret.tlsConfig.RootCAs = x509.NewCertPool()
			pemdata, err := ioutil.ReadFile(ret.SSLCA)
			if err != nil {
				return nil, fmt.Errorf("Failure reading CA certificate: %s\n", err)
			}
			rest := pemdata
			var block *pem.Block
			var pemBlockNum = 1
			for {
				block, rest = pem.Decode(rest)
				if block != nil {
					if block.Type != "CERTIFICATE" {
						return nil, fmt.Errorf("Block %d does not contain a certificate: %s\n", pemBlockNum, ret.SSLCA)
					}
					cert, err := x509.ParseCertificate(block.Bytes)
					if err != nil {
						return nil, fmt.Errorf("Failed to parse CA certificate in block %d: %s\n", pemBlockNum, ret.SSLCA)
					}
					ret.tlsConfig.RootCAs.AddCert(cert)
					pemBlockNum++
				} else {
					break
				}
			}
		}
	} else {
		if err := config.ReportUnusedConfig(configPath, unused); err != nil {
			return nil, err
		}
	}

	return ret, nil
}

// NewTransport returns a new Transport interface using the settings from the
// TransportTCPFactory.
func (f *TransportTCPFactory) NewTransport(observer transports.Observer) transports.Transport {
	ret := &TransportTCP{
		config:         f,
		observer:       observer,
		controllerChan: make(chan int),
	}

	go ret.controller()

	return ret
}

// Register the transports
func init() {
	config.RegisterTransport("tcp", NewTransportTCPFactory)
	config.RegisterTransport("tls", NewTransportTCPFactory)
}
