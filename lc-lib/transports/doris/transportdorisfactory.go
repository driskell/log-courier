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
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports"
)

const (
	defaultRoutines               int           = 4
	defaultRetry                  time.Duration = 0 * time.Second
	defaultRetryMax               time.Duration = 300 * time.Second
	defaultDatabase               string        = "default"
	defaultRestJSONColumn         string        = "rest"
	defaultPartitionDays          int           = 1
	defaultPartitionRetentionDays int           = 90
)

var (
	// TransportDoris is the transport name for Doris HTTP
	TransportDoris = "doris"
	// TransportDorisHTTPS is the transport name for Doris HTTPS
	TransportDorisHTTPS = "doris-https"
)

// TransportDorisFactory holds the configuration from the configuration file
// It allows creation of TransportDoris instances that use this configuration
type TransportDorisFactory struct {
	// Constructor
	config    *config.Config
	transport string

	// Configuration
	Database               string            `config:"database"`
	TablePattern           string            `config:"table pattern"`
	MetadataServers        []string          `config:"metadata servers"`
	AdditionalColumns      []string          `config:"additional columns"`
	RestJSONColumn         string            `config:"rest json column"`
	Password               string            `config:"password"`
	Retry                  time.Duration     `config:"retry backoff"`
	RetryMax               time.Duration     `config:"retry backoff max"`
	Routines               int               `config:"routines"`
	Username               string            `config:"username"`
	LoadProperties         map[string]string `config:"load properties"`
	PartitionDays          int               `config:"partition days"`
	PartitionRetentionDays int               `config:"partition retention days"`

	// Internal - parsed column definitions
	additionalColumnDefs map[string]string
	metadataEntries      []*addresspool.PoolEntry

	*transports.ClientTlsConfiguration `config:",embed"`
}

// NewTransportDorisFactory create a new TransportDorisFactory from the provided
// configuration data, reporting back any configuration errors it discovers
func NewTransportDorisFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (transports.TransportFactory, error) {
	ret := &TransportDorisFactory{
		config:    p.Config(),
		transport: name,
	}
	if err := p.Populate(ret, unUsed, configPath, true); err != nil {
		return nil, err
	}
	return ret, nil
}

// Validate the configuration
func (f *TransportDorisFactory) Validate(p *config.Parser, configPath string) (err error) {
	if f.Routines < 1 {
		return fmt.Errorf("%sroutines cannot be less than 1", configPath)
	}
	if f.Routines > 32 {
		return fmt.Errorf("%sroutines cannot be more than 32", configPath)
	}

	if f.Database == "" {
		return fmt.Errorf("%sdatabase is required", configPath)
	}

	if f.TablePattern == "" {
		return fmt.Errorf("%stable pattern is required", configPath)
	}

	if len(f.MetadataServers) != 0 {
		// Validate metadata servers uniqueness
		metadataServers := make(map[string]bool)
		for _, server := range f.MetadataServers {
			if _, exists := metadataServers[server]; exists {
				return fmt.Errorf("%smetadata servers must be unique: %s appears multiple times", configPath, server)
			}
			metadataServers[server] = true
		}
	}

	if f.RestJSONColumn == "" {
		return fmt.Errorf("%srest json column is required", configPath)
	}

	if f.PartitionRetentionDays < 1 {
		return fmt.Errorf("%spartition retention days cannot be less than 1", configPath)
	}

	// Note: PartitionDays is reserved for future use to support multi-day partitions
	// Currently only daily partitions are supported
	if f.PartitionDays != 1 {
		return fmt.Errorf("%spartition days must be 1 (only daily partitions are currently supported)", configPath)
	}

	// Parse additional columns and their types
	f.additionalColumnDefs = make(map[string]string)
	for _, col := range f.AdditionalColumns {
		parts := strings.Split(col, ":")
		if len(parts) == 1 {
			// No type specified, default to STRING
			f.additionalColumnDefs[parts[0]] = "STRING"
		} else if len(parts) == 2 {
			colName := parts[0]
			colType := strings.ToUpper(parts[1])
			// Validate type
			validTypes := map[string]bool{
				"STRING": true, "INT": true, "BIGINT": true, "DOUBLE": true,
				"FLOAT": true, "BOOLEAN": true, "DATE": true, "DATETIME": true,
				"JSON": true,
			}
			if !validTypes[colType] {
				return fmt.Errorf("%sadditional columns: invalid type '%s' for column '%s'", configPath, parts[1], colName)
			}
			f.additionalColumnDefs[colName] = colType
		} else {
			return fmt.Errorf("%sadditional columns: invalid format '%s', expected 'name' or 'name:type'", configPath, col)
		}
	}

	return f.ClientTlsConfiguration.TlsValidate(f.transport == TransportDorisHTTPS, p, configPath)
}

// Defaults sets the default configuration values
func (f *TransportDorisFactory) Defaults() {
	f.Routines = defaultRoutines
	f.Retry = defaultRetry
	f.RetryMax = defaultRetryMax
	f.Database = defaultDatabase
	f.RestJSONColumn = defaultRestJSONColumn
	f.PartitionDays = defaultPartitionDays
	f.PartitionRetentionDays = defaultPartitionRetentionDays
	f.LoadProperties = make(map[string]string)
}

// NewTransport returns a new Transport interface using the settings from the
// TransportDorisFactory.
func (f *TransportDorisFactory) NewTransport(ctx context.Context, poolEntry *addresspool.PoolEntry, eventChan chan<- transports.Event) transports.Transport {
	ctx, shutdownFunc := context.WithCancel(ctx)

	ret := &transportDoris{
		ctx:          ctx,
		shutdownFunc: shutdownFunc,
		config:       f,
		netConfig:    transports.FetchConfig(f.config),
		poolEntry:    poolEntry,
		eventChan:    eventChan,
		clientCache:  make(map[string]*clientCacheItem),
		tablePattern: event.NewPatternFromString(f.TablePattern),
	}

	ret.startController()
	return ret
}

// ShouldRestart returns true if the transport needs to be restarted in order
// for the new configuration to apply
func (t *TransportDorisFactory) ShouldRestart(newConfig transports.TransportFactory) bool {
	newConfigImpl := newConfig.(*TransportDorisFactory)
	if newConfigImpl.Database != t.Database {
		return true
	}
	if newConfigImpl.TablePattern != t.TablePattern {
		return true
	}
	if !reflect.DeepEqual(newConfigImpl.MetadataServers, t.MetadataServers) {
		return true
	}
	if !reflect.DeepEqual(newConfigImpl.AdditionalColumns, t.AdditionalColumns) {
		return true
	}
	if newConfigImpl.RestJSONColumn != t.RestJSONColumn {
		return true
	}
	if newConfigImpl.Password != t.Password {
		return true
	}
	if newConfigImpl.Retry != t.Retry {
		return true
	}
	if newConfigImpl.RetryMax != t.RetryMax {
		return true
	}
	if newConfigImpl.Routines != t.Routines {
		return true
	}
	if newConfigImpl.Username != t.Username {
		return true
	}
	if !reflect.DeepEqual(newConfigImpl.LoadProperties, t.LoadProperties) {
		return true
	}
	if newConfigImpl.PartitionDays != t.PartitionDays {
		return true
	}
	if newConfigImpl.PartitionRetentionDays != t.PartitionRetentionDays {
		return true
	}

	return t.ClientTlsConfiguration.HasChanged(newConfigImpl.ClientTlsConfiguration)
}

// Register the transports
func init() {
	transports.RegisterTransport(TransportDoris, NewTransportDorisFactory)
	transports.RegisterTransport(TransportDorisHTTPS, NewTransportDorisFactory)
}
