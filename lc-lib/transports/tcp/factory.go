/*
 * Copyright 2012-2023 Jason Woods and contributors
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
	"regexp"

	"github.com/driskell/log-courier/lc-lib/config"
)

// Factory holds common TCP factory settings
type Factory struct {
	config         *config.Config
	hostportRegexp *regexp.Regexp
}

func newFactory(config *config.Config) *Factory {
	return &Factory{
		config:         config,
		hostportRegexp: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
	}
}
