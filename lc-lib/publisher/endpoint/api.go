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

package endpoint

import (
	"time"

	"github.com/driskell/log-courier/lc-lib/admin/api"
)

type apiEndpoint struct {
	api.KeyValue

	e *Endpoint
}

// Update the status information for an endpoint
func (a *apiEndpoint) Update() error {
	a.e.mutex.RLock()
	a.SetEntry("server", api.String(a.e.server))
	a.SetEntry("status", api.String(a.e.status.String()))
	if a.e.lastErr != nil {
		a.SetEntry("last_error", api.String(a.e.lastErr.Error()))
		a.SetEntry("last_error_time", api.String(a.e.lastErrTime.Format(time.RFC3339)))
	} else {
		a.SetEntry("last_error", api.Null)
		a.SetEntry("last_error_time", api.Null)
	}
	a.SetEntry("pendingPayloads", api.Number(a.e.NumPending()))
	a.SetEntry("publishedLines", api.Number(a.e.LineCount()))
	a.SetEntry("averageLatency", api.Float(a.e.AverageLatency()/time.Millisecond))
	a.e.mutex.RUnlock()

	return nil
}
