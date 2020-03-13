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
	if err != nil {
		event.AddError("user_agent", fmt.Sprintf("Field lookup failed: %s", err))
		return event
	}

	var (
		value string
		ok    bool
	)
	if value, ok = entry.(string); !ok {
		event.AddError("user_agent", fmt.Sprintf("Field '%s' is not present", g.Field))
		return event
	}

	var client *uaparser.Client
	if cachedClient, ok := g.lru.Get(value); ok {
		client = cachedClient.(*uaparser.Client)
	} else {
		client = g.parser.Parse(value)
		g.lru.Add(value, client)
	}

	var data map[string]interface{}
	if data, ok = event.MustResolve("user_agent", nil).(map[string]interface{}); !ok {
		data = map[string]interface{}{}
	}

	data["user_agent[name]"] = client.UserAgent.Family
	if client.Device.Family != "" {
		data["user_agent[device][name]"] = client.Device.Family
	}
	if versionString := client.UserAgent.ToVersionString(); versionString != "" {
		data["user_agent[major]"] = versionString
	}
	if client.UserAgent.Major != "" {
		data["user_agent[major]"] = client.UserAgent.Major
	}
	if client.UserAgent.Minor != "" {
		data["user_agent[minor]"] = client.UserAgent.Minor
	}
	if client.UserAgent.Patch != "" {
		data["user_agent[patch]"] = client.UserAgent.Patch
	}
	if client.Os.Family != "" {
		data["user_agent[os][family]"] = client.Os.Family
	}
	if versionString := client.Os.ToVersionString(); versionString != "" {
		data["user_agent[os][family]"] = versionString
	}
	if client.Os.Major != "" {
		data["user_agent[os][major]"] = client.Os.Major
	}
	if client.Os.Minor != "" {
		data["user_agent[os][minor]"] = client.Os.Minor
	}
	if client.Os.PatchMinor != "" {
		data["user_agent[os][version]"] = client.Os.PatchMinor
	}

	event.Resolve("user_agent", data)
	return event
}

// init will register the action
func init() {
	RegisterAction("user_agent", newUserAgentAction)
}
