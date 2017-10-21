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
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/transports"
)

var (
	// TransportTCPTCP is the transport name for plain TCP
	TransportTCPTCP = "tcp"
	// TransportTCPTLS is the transport name for encrypted TLS
	TransportTCPTLS = "tls"
)

const (
	defaultNetworkReconnect    time.Duration = 0 * time.Second
	defaultNetworkReconnectMax time.Duration = 300 * time.Second
)

// TransportTCPFactory holds the configuration from the configuration file
// It allows creation of TransportTCP instances that use this configuration
type TransportTCPFactory struct {
	transport string

	config *config.Config

	Reconnect      time.Duration `config:"reconnect backoff"`
	ReconnectMax   time.Duration `config:"reconnect backoff max"`
	SSLCertificate string        `config:"ssl certificate"`
	SSLKey         string        `config:"ssl key"`
	SSLCA          string        `config:"ssl ca"`

	hostportRegexp  *regexp.Regexp
	certificate     *tls.Certificate
	certificateList []*x509.Certificate
	caList          []*x509.Certificate
}

// NewTransportTCPFactory create a new TransportTCPFactory from the provided
// configuration data, reporting back any configuration errors it discovers.
func NewTransportTCPFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (interface{}, error) {
	var err error

	ret := &TransportTCPFactory{
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

		if len(ret.SSLCA) == 0 {
			return nil, errors.New("ssl ca is required when transport is TLS")
		}

		pemdata, err := ioutil.ReadFile(ret.SSLCA)
		if err != nil {
			return nil, fmt.Errorf("failure reading CA certificate: %s\n", err)
		}
		rest := pemdata
		var block *pem.Block
		var pemBlockNum = 1
		for {
			block, rest = pem.Decode(rest)
			if block != nil {
				if block.Type != "CERTIFICATE" {
					return nil, fmt.Errorf("block %d does not contain a certificate: %s\n", pemBlockNum, ret.SSLCA)
				}
				cert, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					return nil, fmt.Errorf("failed to parse CA certificate in block %d: %s\n", pemBlockNum, ret.SSLCA)
				}
				ret.caList = append(ret.caList, cert)
				pemBlockNum++
			} else {
				break
			}
		}
	} else {
		if len(ret.SSLCertificate) > 0 || len(ret.SSLKey) > 0 || len(ret.SSLCA) > 0 {
			return nil, fmt.Errorf("transport tcp does not support ssl certificate, ssl key or ssl ca configurations")
		}
	}

	return ret, nil
}

// Defaults sets the default configuration values
func (f *TransportTCPFactory) Defaults() {
	f.Reconnect = defaultNetworkReconnect
	f.ReconnectMax = defaultNetworkReconnectMax
}

// NewTransport returns a new Transport interface using the settings from the
// TransportTCPFactory.
func (f *TransportTCPFactory) NewTransport(observer transports.Observer, finishOnFail bool) transports.Transport {
	ret := &TransportTCP{
		config:         f,
		netConfig:      transports.FetchConfig(f.config),
		finishOnFail:   finishOnFail,
		observer:       observer,
		controllerChan: make(chan int),
		backoff:        core.NewExpBackoff(observer.Pool().Server()+" Reconnect", f.Reconnect, f.ReconnectMax),
	}

	go ret.controller()

	return ret
}

// Register the transports
func init() {
	transports.Register(TransportTCPTCP, NewTransportTCPFactory)
	transports.Register(TransportTCPTLS, NewTransportTCPFactory)
}
