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

package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// loadYAMLFile loads the given YAML format file
func loadYAMLFile(path string, rawConfig interface{}) (err error) {
	var data []byte

	// Read the entire file
	if data, err = ioutil.ReadFile(path); err != nil {
		return
	}

	// Pull the entire structure into rawConfig
	err = yaml.Unmarshal(data, rawConfig)
	return
}
