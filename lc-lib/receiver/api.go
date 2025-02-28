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

package receiver

import (
	"github.com/driskell/log-courier/lc-lib/admin/api"
)

type apiStatus struct {
	api.KeyValue

	r *Pool
}

// Update updates the prospector status information
func (a *apiStatus) Update() error {
	// Update the values and pass through to node
	a.r.connectionLock.RLock()
	a.SetEntry("activeConnections", api.Number(len(a.r.connectionStatus)))
	a.SetEntry("queuePayloads", api.Number(len(a.r.spool)))
	a.SetEntry("queueSize", api.Number(a.r.spoolSize))
	a.r.connectionLock.RUnlock()

	return nil
}
