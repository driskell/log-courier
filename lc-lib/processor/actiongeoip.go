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

package processor

import (
	"fmt"
	"net"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/hashicorp/golang-lru/simplelru"
	"github.com/oschwald/geoip2-golang"
)

type geoIPAction struct {
	Field    string `config:"field"`
	Database string `config:"database"`

	lru    simplelru.LRUCache
	reader *geoip2.Reader
}

type geoipActionLookupResult struct {
	record *geoip2.City
	err    error
}

func newGeoIPAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (Action, error) {
	var err error
	action := &geoIPAction{}
	if err = p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	action.lru, err = simplelru.NewLRU(1000, nil)
	if err != nil {
		return nil, err
	}
	action.reader, err = geoip2.Open(action.Database)
	if err != nil {
		return nil, err
	}
	return action, nil
}

func (g *geoIPAction) Process(event *event.Event) *event.Event {
	entry, err := event.Resolve(g.Field, nil)
	if value, ok := entry.(string); err != nil && ok {
		var result *geoipActionLookupResult
		if cachedRecord, ok := g.lru.Get(value); ok {
			result = cachedRecord.(*geoipActionLookupResult)
		} else {
			ip := net.ParseIP(value)
			if ip == nil {
				event.Resolve("_geoip_error", fmt.Sprintf("Field '%s' is not a valid IP address", g.Field))
				event.AddTag("_geoip_failure")
				return event
			}
			record, err := g.reader.City(ip)
			result = &geoipActionLookupResult{record, err}
			g.lru.Add(value, result)
		}
		if result.err != nil {
			event.Resolve("_geoip_error", fmt.Sprintf("GeoIP2 lookup failed: %s", result.err))
			event.AddTag("_geoip_failure")
			return event
		}
		record := result.record
		if record.City.GeoNameID == 0 {
			// Not found, ignore
			return event
		}
		dataValue, _ := event.Resolve("source[geo]", nil)
		var data map[string]interface{}
		var ok bool
		if data, ok = dataValue.(map[string]interface{}); !ok {
			data = map[string]interface{}{}
		}
		data["city_name"] = record.City.Names["en"]
		data["continent_name"] = record.Continent.Names["en"]
		data["country_iso_code"] = record.Country.IsoCode
		data["country_name"] = record.Country.Names["en"]
		data["location"] = map[string]interface{}{
			"type":        "Point",
			"coordinates": []float64{record.Location.Longitude, record.Location.Latitude},
		}
		data["postal_code"] = record.Postal.Code
		data["timezone"] = record.Location.TimeZone
		if len(record.Subdivisions) > 0 {
			data["region_iso_code"] = record.Subdivisions[0].IsoCode
			data["region_name"] = record.Subdivisions[0].Names["en"]
		}
		event.Resolve("source[geo]", data)
	} else {
		event.Resolve("_geoip_error", fmt.Sprintf("Field '%s' is not present", g.Field))
		event.AddTag("_geoip_failure")
	}
	return event
}
