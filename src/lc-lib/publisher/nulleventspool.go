/*
 * Copyright 2014 Jason Woods.
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

package publisher

import (
	"github.com/driskell/log-courier/src/lc-lib/registrar"
)

// NullEventSpool is a dummy registrar used by publisher when there is no
// registrar to send acknowledgements to for persistence.
// It simply discards all acknowledgement requests it is given.
// Currently used during stdin pipe processing where persistence becomes
// irrelevant.
type NullEventSpool struct {
}

// newNullEventSpool creates a new dummy registrar
func newNullEventSpool() *NullEventSpool {
	return &NullEventSpool{}
}

// Close does nothing - it's a dummy registrar
func (s *NullEventSpool) Close() {
}
// Add does nothing - it's a dummy registrar
func (s *NullEventSpool) Add(event registrar.EventProcessor) {
}

// Send does nothing - it's a dummy registrar
func (s *NullEventSpool) Send() {
}
