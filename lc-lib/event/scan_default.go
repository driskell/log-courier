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

// scanTimestamp reports no @timestamp, as without encoding/json/v2 it is found by decoding the event
func scanTimestamp(encoded []byte) (Timestamp, bool) {
	return Timestamp{}, false
}

// encodedNormalised reports the encoded form must be decoded, as without encoding/json/v2 it cannot be checked cheaply
func encodedNormalised(encoded []byte) bool {
	return false
}
