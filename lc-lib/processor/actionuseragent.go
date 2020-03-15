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

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	lru "github.com/hashicorp/golang-lru"
	"github.com/ua-parser/uap-go/uaparser"
)

type userAgentAction struct {
	Field  string `config:"field"`
	Remove bool   `config:"remove"`

	lru    *lru.Cache
	parser *uaparser.Parser
}

func newUserAgentAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (ASTEntry, error) {
	var err error
	action := &userAgentAction{}
	if err = p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	action.lru, err = lru.New(1000)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialise LRU cache for user_agent at %s: %s", configPath, err)
	}
	action.parser = uaparser.NewFromSaved()
	return action, nil
}

func (g *userAgentAction) Process(subject *event.Event) *event.Event {
	entry, err := subject.Resolve(g.Field, nil)
	if err != nil {
		subject.AddError("user_agent", fmt.Sprintf("Field lookup failed: %s", err))
		return subject
	}

	var (
		value string
		ok    bool
	)
	if value, ok = entry.(string); !ok {
		subject.AddError("user_agent", fmt.Sprintf("Field '%s' is not present", g.Field))
		return subject
	}

	var client *uaparser.Client
	if cachedClient, ok := g.lru.Get(value); ok {
		client = cachedClient.(*uaparser.Client)
	} else {
		client = g.parser.Parse(value)
		g.lru.Add(value, client)
	}

	subject.MustResolve("user_agent[original]", value)
	subject.MustResolve("user_agent[name]", client.UserAgent.Family)
	if client.Device.Family != "" {
		subject.MustResolve("user_agent[device][name]", client.Device.Family)
	}
	if versionString := client.UserAgent.ToVersionString(); versionString != "" {
		subject.MustResolve("user_agent[major]", versionString)
	}
	if client.UserAgent.Major != "" {
		subject.MustResolve("user_agent[major]", client.UserAgent.Major)
	}
	if client.UserAgent.Minor != "" {
		subject.MustResolve("user_agent[minor]", client.UserAgent.Minor)
	}
	if client.UserAgent.Patch != "" {
		subject.MustResolve("user_agent[patch]", client.UserAgent.Patch)
	}
	if client.Os.Family != "" {
		subject.MustResolve("user_agent[os][family]", client.Os.Family)
	}
	if versionString := client.Os.ToVersionString(); versionString != "" {
		subject.MustResolve("user_agent[os][family]", versionString)
	}
	if client.Os.Major != "" {
		subject.MustResolve("user_agent[os][major]", client.Os.Major)
	}
	if client.Os.Minor != "" {
		subject.MustResolve("user_agent[os][minor]", client.Os.Minor)
	}
	if client.Os.PatchMinor != "" {
		subject.MustResolve("user_agent[os][version]", client.Os.PatchMinor)
	}
	if g.Remove {
		_, err := subject.Resolve(g.Field, event.ResolveParamUnset)
		if err != nil {
			subject.AddError("user_agent", fmt.Sprintf("Failed to remove field '%s': %s", g.Field, err))
		}
	}
	return subject
}

// init will register the action
func init() {
	RegisterAction("user_agent", newUserAgentAction)
}
