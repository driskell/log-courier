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
)

var (
	// TransportES is the transport name for ES HTTP
	TransportES = "es"

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
	Routines         int           `config:"routines"`
	Retry            time.Duration `config:"retry backoff"`
	RetryMax         time.Duration `config:"retry backoff max"`
	IndexPattern     string        `config:"index pattern"`
	TemplateFile     string        `config:"template file"`
	TemplatePatterns []string      `config:"template patterns"`

	// Internal
	template []byte
}

// NewTransportESFactory create a new TransportESFactory from the provided
// configuration data, reporting back any configuration errors it discovers
func NewTransportESFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (transports.TransportFactory, error) {
	var err error

	ret := &TransportESFactory{
		config:    p.Config(),
		transport: name,
	}

	if err = p.Populate(ret, unUsed, configPath, true); err != nil {
		return nil, err
	}

	if ret.IndexPattern == "" {
		return nil, fmt.Errorf("'index pattern' cannot be empty when using 'es' transport")
	}

	if ret.TemplateFile != "" {
		file, err := os.Open(ret.TemplateFile)
		if err != nil {
			return nil, err
		}
		defer func() {
			file.Close()
		}()
		ret.template, err = ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
	} else {
		if len(ret.TemplatePatterns) == 0 {
			return nil, fmt.Errorf("'template patterns' is required when 'template file' is not set when using 'es' transport")
		}

		var result []byte
		result, err = json.Marshal(ret.TemplatePatterns)
		if err != nil {
			panic(fmt.Sprintf("'template patterns' failed to encode: %s", err))
		}
		ret.templatePatternsJSON = string(result)

		result, err = json.Marshal(ret.TemplatePatterns[0])
		if err != nil {
			panic(fmt.Sprintf("'template patterns' failed to encode: %s", err))
		}
		ret.templatePatternSingleJSON = string(result)
	}

	return ret, nil
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
func (f *TransportESFactory) NewTransport(ctx context.Context, pool *addresspool.Pool, eventChan chan<- transports.Event, finishOnFail bool) transports.Transport {
	ctx, shutdownFunc := context.WithCancel(ctx)

	ret := &transportES{
		ctx:          ctx,
		shutdownFunc: shutdownFunc,
		config:       f,
		netConfig:    transports.FetchConfig(f.config),
		finishOnFail: finishOnFail,
		pool:         pool,
		eventChan:    eventChan,
	}

	ret.startController()
	return ret
}

// Register the transports
func init() {
	transports.RegisterTransport(TransportES, NewTransportESFactory)
}
