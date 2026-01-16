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

package processor

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/geoipupdate"
	lru "github.com/hashicorp/golang-lru"
	"github.com/oschwald/geoip2-golang"
)

var (
	// DefaultGeoIPActionDatabase is the default path to the GeoIP database to use
	// It can be changed during init()
	// This is legacy and a geoipupdate is now built in
	DefaultGeoIPActionDatabase = ""
)

const (
	defaultGeoIPActionTarget = "source[geo]"
)

type geoIPAction struct {
	Field    string `config:"field"`
	Database string `config:"database"`
	Target   string `config:"target"`

	config          *config.Config
	lru             *lru.Cache
	reader          *geoip2.Reader
	databaseModTime time.Time
}

type geoipActionLookupResult struct {
	record *geoip2.City
	err    error
}

func newGeoIPAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (ASTEntry, error) {
	var err error
	action := &geoIPAction{
		config: p.Config(),
	}
	if err = p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	return action, nil
}

func (g *geoIPAction) Defaults() {
	g.Target = defaultGeoIPActionTarget
}

func (g *geoIPAction) Validate(p *config.Parser, configPath string) (err error) {
	g.lru, err = lru.New(1000)
	if err != nil {
		return fmt.Errorf("Failed to initialse GeoIP at %s: %s", configPath, err)
	}
	if g.Database != "" {
		// Ensure database is accessible now
		if err = g.openDatabase(); err != nil {
			return fmt.Errorf("Failed to initialse GeoIP at %s: %s", configPath, err)
		}
	}
	return nil
}

func (g *geoIPAction) Process(event *event.Event) *event.Event {
	entry, err := event.Resolve(g.Field, nil)
	if err != nil {
		event.AddError("geoip", fmt.Sprintf("Field lookup failed: %s", err))
		return event
	}

	var (
		value string
		ok    bool
	)
	if value, ok = entry.(string); !ok {
		event.AddError("geoip", fmt.Sprintf("Field '%s' is not present", g.Field))
		return event
	}

	var result *geoipActionLookupResult
	if cachedRecord, ok := g.lru.Get(value); ok {
		result = cachedRecord.(*geoipActionLookupResult)
	} else {
		if g.reader == nil {
			err = g.openDatabase()
			if err != nil {
				event.AddError("geoip", fmt.Sprintf("Failed to open GeoIP2 database: %s", err))
				return event
			}
		} else {
			g.refreshDatabaseIfNeeded()
		}

		ip := net.ParseIP(value)
		if ip == nil {
			event.AddError("geoip", fmt.Sprintf("Field '%s' is not a valid IP address", g.Field))
			return event
		}
		record, err := g.reader.City(ip)
		result = &geoipActionLookupResult{record, err}
		g.lru.Add(value, result)
	}

	if result.err != nil {
		event.AddError("geoip", fmt.Sprintf("GeoIP2 lookup failed: %s", result.err))
		return event
	}

	record := result.record
	if record.City.GeoNameID == 0 {
		// Not found, ignore
		return event
	}

	var data map[string]interface{}
	target, err := event.Resolve(g.Target, nil)
	if err != nil {
		event.AddError("geoip", fmt.Sprintf("Failed to load target field '%s': %s", g.Target, err))
		return event
	}
	if data, ok = target.(map[string]interface{}); !ok {
		data = map[string]interface{}{}
	}

	data["city_name"] = record.City.Names["en"]
	data["continent_name"] = record.Continent.Names["en"]
	data["country_iso_code"] = record.Country.IsoCode
	data["country_name"] = record.Country.Names["en"]
	data["location"] = []float64{record.Location.Longitude, record.Location.Latitude}
	data["latitude"] = record.Location.Latitude
	data["longitude"] = record.Location.Longitude
	data["postal_code"] = record.Postal.Code
	data["timezone"] = record.Location.TimeZone
	if len(record.Subdivisions) > 0 {
		data["region_iso_code"] = record.Subdivisions[0].IsoCode
		data["region_name"] = record.Subdivisions[0].Names["en"]
	}

	if _, err := event.Resolve(g.Target, data); err != nil {
		event.AddError("geoip", fmt.Sprintf("Failed to set target field '%s': %s", g.Target, err))
		return event
	}
	return event
}

// openDatabase opens (or re-opens) the GeoIP2 database
func (g *geoIPAction) openDatabase() error {
	database := g.Database
	accountId := g.config.GeneralPart("geoipupdate").(*geoipupdate.General).AccountId
	if database == "" && accountId != 0 {
		database = geoipupdate.GetDatabasePath(g.config)
	}
	fileStat, err := os.Stat(database)
	if err != nil {
		return fmt.Errorf("GeoIP database file '%s' is not accessible: %s", database, err)
	}
	g.databaseModTime = fileStat.ModTime()
	g.reader, err = geoip2.Open(database)
	if err != nil {
		return fmt.Errorf("Failed to initialse GeoIP: %s", err)
	}
	return nil
}

// refreshDatabaseIfNeeded refreshes the database reader if the database file has changed
func (g *geoIPAction) refreshDatabaseIfNeeded() {
	database := g.Database
	accountId := g.config.GeneralPart("geoipupdate").(*geoipupdate.General).AccountId
	if database == "" && accountId != 0 {
		database = geoipupdate.GetDatabasePath(g.config)
	}
	fileStat, err := os.Stat(database)
	if err != nil {
		log.Errorf("Failed to reopen GeoIP database: %s", err)
	}
	if fileStat.ModTime().After(g.databaseModTime) {
		// Database has changed, re-open
		oldReader := g.reader
		g.reader = nil
		if err = g.openDatabase(); err != nil {
			log.Errorf("Failed to reopen GeoIP database: %s", err)
			g.reader = oldReader
			return
		}
		log.Infof("Reopened updated GeoIP database '%s'", database)
		oldReader.Close()
		g.lru.Purge()
	}
}

// init will register the action
func init() {
	RegisterAction("geoip", newGeoIPAction)
}
