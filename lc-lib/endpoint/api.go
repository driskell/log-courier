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

import "github.com/driskell/log-courier/lc-lib/admin"

type apiEndpoint struct {
	admin.APIKeyValue

	e *Endpoint
}

// Update the status information for an endpoint
func (a *apiEndpoint) Update() error {
	a.e.mutex.RLock()
	a.SetEntry("server", admin.APIString(a.e.server))
	a.SetEntry("status", admin.APIString(a.e.status.String()))
	a.SetEntry("pendingPayloads", admin.APINumber(a.e.NumPending()))
	a.SetEntry("publishedLines", admin.APINumber(a.e.LineCount()))
	a.e.mutex.RUnlock()

	return nil
}
