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
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/transports"
)

const (
	defaultRoutines     int           = 4
	defaultRetry        time.Duration = 0 * time.Second
	defaultRetryMax     time.Duration = 300 * time.Second
	defaultIndexPattern string        = "logstash-%{+2006-01-02}"

	// Default to TLS 1.2 minimum, supported since Go 1.2
	defaultMinTLSVersion = tls.VersionTLS12
	defaultMaxTLSVersion = 0
)

var (
	// TransportES is the transport name for ES HTTP
	TransportES = "es"
	// TransportESHTTPS is the transport name for ES HTTPS
	TransportESHTTPS = "es-https"

	defaultTemplatePatterns []string = []string{"logstash-*"}
)

// TransportESFactory holds the configuration from the configuration file
// It allows creation of TransportES instances that use this configuration
type TransportESFactory struct {
	// Constructor
	config                    *config.Config
	transport                 string
	templatePatternsJSON      string
	templatePatternSingleJSON string

	// Configuration
	MinTLSVersion    string        `config:"min tls version"`
	MaxTLSVersion    string        `config:"max tls version"`
	IndexPattern     string        `config:"index pattern"`
	Password         string        `config:"password"`
	Retry            time.Duration `config:"retry backoff"`
	RetryMax         time.Duration `config:"retry backoff max"`
	Routines         int           `config:"routines"`
	SSLCA            string        `config:"ssl ca"`
	Username         string        `config:"username"`
	TemplateFile     string        `config:"template file"`
	TemplatePatterns []string      `config:"template patterns"`

	// Internal
	template      []byte
	caList        []*x509.Certificate
	minTLSVersion uint16
	maxTLSVersion uint16
}

// NewTransportESFactory create a new TransportESFactory from the provided
// configuration data, reporting back any configuration errors it discovers
func NewTransportESFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (transports.TransportFactory, error) {
	ret := &TransportESFactory{
		config:    p.Config(),
		transport: name,
	}
	if err := p.Populate(ret, unUsed, configPath, true); err != nil {
		return nil, err
	}
	return ret, nil
}

// Validate the configuration
func (f *TransportESFactory) Validate(p *config.Parser, configPath string) (err error) {
	if f.Routines < 1 {
		return fmt.Errorf("%sroutines cannot be less than 1", configPath)
	}
	if f.Routines > 32 {
		return fmt.Errorf("%sroutines cannot be more than 32", configPath)
	}

	if f.IndexPattern == "" {
		return fmt.Errorf("%sindex pattern is required", configPath)
	}

	if f.TemplateFile != "" {
		var file *os.File
		file, err = os.Open(f.TemplateFile)
		if err != nil {
			return
		}
		defer func() {
			file.Close()
		}()
		f.template, err = ioutil.ReadAll(file)
		if err != nil {
			return
		}
	} else {
		if len(f.TemplatePatterns) == 0 {
			return fmt.Errorf("%[1]stemplate patterns is required when %[1]stemplate file is not set", configPath)
		}

		var result []byte
		result, err = json.Marshal(f.TemplatePatterns)
		if err != nil {
			panic(fmt.Sprintf("%stemplate patterns failed to encode: %s", configPath, err))
		}
		f.templatePatternsJSON = string(result)

		result, err = json.Marshal(f.TemplatePatterns[0])
		if err != nil {
			panic(fmt.Sprintf("%stemplate patterns failed to encode: %s", configPath, err))
		}
		f.templatePatternSingleJSON = string(result)
	}

	if f.transport == TransportESHTTPS {
		// Check tls versions
		f.minTLSVersion, err = transports.ParseTLSVersion(f.MinTLSVersion, defaultMinTLSVersion)
		if err != nil {
			return
		}
		f.maxTLSVersion, err = transports.ParseTLSVersion(f.MaxTLSVersion, defaultMaxTLSVersion)
		if err != nil {
			return
		}
		// Check SSLCA
		if f.caList, err = transports.AddCertificates(f.caList, f.SSLCA); err != nil {
			return fmt.Errorf("failure loading %sssl ca: %s", configPath, err)
		}
	} else {
		if len(f.MinTLSVersion) > 0 || len(f.MaxTLSVersion) > 0 {
			return fmt.Errorf("%[1]smin tls version and %[1]smax tls version are not supported when the transport is es", configPath)
		}
		if len(f.SSLCA) > 0 {
			return fmt.Errorf("%[1]sssl ca is not supported when the transport is es", configPath)
		}
	}

	return nil
}

// Defaults sets the default configuration values
func (f *TransportESFactory) Defaults() {
	f.Routines = defaultRoutines
	f.Retry = defaultRetry
	f.RetryMax = defaultRetryMax
	f.IndexPattern = defaultIndexPattern
	f.TemplatePatterns = defaultTemplatePatterns
}

// NewTransport returns a new Transport interface using the settings from the
// TransportTCPFactory.
func (f *TransportESFactory) NewTransport(ctx context.Context, pool *addresspool.Pool, eventChan chan<- transports.Event) transports.Transport {
	ctx, shutdownFunc := context.WithCancel(ctx)

	ret := &transportES{
		ctx:          ctx,
		shutdownFunc: shutdownFunc,
		config:       f,
		netConfig:    transports.FetchConfig(f.config),
		pool:         pool,
		eventChan:    eventChan,
	}

	ret.startController()
	return ret
}

// Register the transports
func init() {
	transports.RegisterTransport(TransportES, NewTransportESFactory)
	transports.RegisterTransport(TransportESHTTPS, NewTransportESFactory)
}
