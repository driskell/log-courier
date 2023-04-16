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

package action

import (
	"fmt"
	"net"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
	lru "github.com/hashicorp/golang-lru"
	"github.com/oschwald/geoip2-golang"
)

var (
	// DefaultGeoIPNodeDatabase is the default path to the GeoIP database to use
	// It can be changed during init()
	DefaultGeoIPNodeDatabase = ""
)

const (
	defaultGeoIPNodeTarget = "source[geo]"
)

type geoIPNode struct {
	database string
	lru      *lru.Cache
	reader   *geoip2.Reader
}

var _ ast.ProcessArgumentsNode = &geoIPNode{}

type geoipNodeLookupResult struct {
	record *geoip2.City
	err    error
}

func newGeoIPNode() (ast.ProcessArgumentsNode, error) {
	var err error
	node := &geoIPNode{database: DefaultGeoIPNodeDatabase}
	node.lru, err = lru.New(1000)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialse LRU cache for GeoIP: %s", err)
	}
	return node, nil
}

func (n *geoIPNode) Arguments() []ast.Argument {
	databaseOptions := ast.ArgumentOptional & ast.ArgumentExprDisallowed
	if n.database == "" {
		databaseOptions = databaseOptions & ast.ArgumentRequired
	}
	return []ast.Argument{
		ast.NewArgumentString("database", databaseOptions),
		ast.NewArgumentString("field", ast.ArgumentRequired),
		ast.NewArgumentString("target", ast.ArgumentOptional),
	}
}

func (n *geoIPNode) Init(arguments []any) error {
	n.database = arguments[0].(string)
	var err error
	n.reader, err = geoip2.Open(n.database)
	if err != nil {
		return fmt.Errorf("Failed to initialse GeoIP database at %s: %s", n.database, err)
	}
	return nil
}

func (n *geoIPNode) ProcessWithArguments(subject *event.Event, arguments []any) *event.Event {
	field := arguments[1].(string)
	target := defaultGeoIPNodeTarget
	if arguments[2] != nil {
		target = arguments[2].(string)
	}

	entry, err := subject.Resolve(field, nil)
	if err != nil {
		subject.AddError("geoip", fmt.Sprintf("Field lookup failed: %s", err))
		return subject
	}

	var (
		value string
		ok    bool
	)
	if value, ok = entry.(string); !ok {
		subject.AddError("geoip", fmt.Sprintf("Field '%s' is not present", field))
		return subject
	}

	var result *geoipNodeLookupResult
	if cachedRecord, ok := n.lru.Get(value); ok {
		result = cachedRecord.(*geoipNodeLookupResult)
	} else {
		ip := net.ParseIP(value)
		if ip == nil {
			subject.AddError("geoip", fmt.Sprintf("Field '%s' is not a valid IP address", field))
			return subject
		}
		record, err := n.reader.City(ip)
		result = &geoipNodeLookupResult{record, err}
		n.lru.Add(value, result)
	}

	if result.err != nil {
		subject.AddError("geoip", fmt.Sprintf("GeoIP2 lookup failed: %s", result.err))
		return subject
	}

	record := result.record
	if record.City.GeoNameID == 0 {
		// Not found, ignore
		return subject
	}

	var data map[string]interface{}
	targetValue, err := subject.Resolve(target, nil)
	if err != nil {
		subject.AddError("geoip", fmt.Sprintf("Failed to load target field '%s': %s", target, err))
		return subject
	}
	if data, ok = targetValue.(map[string]interface{}); !ok {
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

	if _, err := subject.Resolve(target, data); err != nil {
		subject.AddError("geoip", fmt.Sprintf("Failed to set target field '%s': %s", target, err))
		return subject
	}
	return subject
}

// init will register the action
func init() {
	ast.RegisterAction("geoip", newGeoIPNode)
}
