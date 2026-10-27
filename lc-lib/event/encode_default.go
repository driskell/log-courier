//go:build !go1.27 || !goexperiment.jsonv2

/*
 * Copyright 2026 Jason Woods and contributors
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

package event

import (
	"encoding/json"
	"sync"
)

// encodeMaps holds maps reused to encode events without @metadata, so encoding does not allocate a map per event
var encodeMaps = sync.Pool{
	New: func() interface{} {
		return make(map[string]interface{})
	},
}

// encodeData encodes event data without its @metadata, leaving the data unchanged as it may be read elsewhere meanwhile
func encodeData(data map[string]interface{}) ([]byte, error) {
	prunedData := encodeMaps.Get().(map[string]interface{})
	for key, value := range data {
		if key == MetadataKey {
			continue
		}
		prunedData[key] = value
	}
	encoded, err := json.Marshal(prunedData)
	clear(prunedData)
	encodeMaps.Put(prunedData)
	return encoded, err
}
