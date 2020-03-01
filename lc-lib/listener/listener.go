/*
 * Copyright 2014-2016 Jason Woods.
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

package listener

import (
	"sync"

	"github.com/driskell/log-courier/lc-lib/config"
)

// Listener listens for incoming events and spools them
type Listener struct {
	config *config.Config
}

// Init the listener
func (r *Listener) Init(config *config.Config) error {
	return nil
}

// Run the receiver
func (r *Listener) Run(group *sync.WaitGroup) {
	defer func() {
		group.Done()
	}()
}
