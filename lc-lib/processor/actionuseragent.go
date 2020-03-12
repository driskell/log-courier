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

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/hashicorp/golang-lru/simplelru"
	"github.com/ua-parser/uap-go/uaparser"
)

type userAgentAction struct {
	Field string `config:"field"`

	lru    simplelru.LRUCache
	parser *uaparser.Parser
}

func newUserAgentAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (Action, error) {
	var err error
	action := &userAgentAction{}
	if err = p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	action.lru, err = simplelru.NewLRU(1000, nil)
	if err != nil {
		return nil, err
	}
	action.parser = uaparser.NewFromSaved()
	return action, nil
}

func (g *userAgentAction) Process(event *event.Event) *event.Event {
	entry, err := event.Resolve(g.Field, nil)
	if value, ok := entry.(string); err != nil && ok {
		var client *uaparser.Client
		if cachedClient, ok := g.lru.Get(value); ok {
			client = cachedClient.(*uaparser.Client)
		} else {
			client = g.parser.Parse(value)
			g.lru.Add(value, client)
		}
		userAgentValue, _ := event.Resolve("user_agent", nil)
		var userAgent map[string]interface{}
		var ok bool
		if userAgent, ok = userAgentValue.(map[string]interface{}); !ok {
			userAgent = map[string]interface{}{}
		}
		userAgent["user_agent[name]"] = client.UserAgent.Family
		if client.Device.Family != "" {
			userAgent["user_agent[device][name]"] = client.Device.Family
		}
		if versionString := client.UserAgent.ToVersionString(); versionString != "" {
			userAgent["user_agent[major]"] = versionString
		}
		if client.UserAgent.Major != "" {
			userAgent["user_agent[major]"] = client.UserAgent.Major
		}
		if client.UserAgent.Minor != "" {
			userAgent["user_agent[minor]"] = client.UserAgent.Minor
		}
		if client.UserAgent.Patch != "" {
			userAgent["user_agent[patch]"] = client.UserAgent.Patch
		}
		if client.Os.Family != "" {
			userAgent["user_agent[os][family]"] = client.Os.Family
		}
		if versionString := client.Os.ToVersionString(); versionString != "" {
			userAgent["user_agent[os][family]"] = versionString
		}
		if client.Os.Major != "" {
			userAgent["user_agent[os][major]"] = client.Os.Major
		}
		if client.Os.Minor != "" {
			userAgent["user_agent[os][minor]"] = client.Os.Minor
		}
		if client.Os.PatchMinor != "" {
			userAgent["user_agent[os][version]"] = client.Os.PatchMinor
		}
		event.Resolve("user_agent", userAgent)
	} else {
		event.Resolve("_geoip_error", fmt.Sprintf("Field '%s' is not present", g.Field))
		event.AddTag("_geoip_failure")
	}
	return event
}
