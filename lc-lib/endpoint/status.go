/*
 * Copyright 2014-2015 Jason Woods.
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

package endpoint

// StatusChange holds a value that represents a change in endpoint status that
// is sent over the status channel of the Sink
type StatusChange int

// Endpoint status signals
const (
	Ready = iota
	Recovered
	Failed
	Finished
)

// Status structure contains the reason for failure, or nil if recovered
type Status struct {
	Endpoint *Endpoint
	Status   StatusChange
}
