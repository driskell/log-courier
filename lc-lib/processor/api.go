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

package processor

import (
	"time"

	"github.com/driskell/log-courier/lc-lib/admin/api"
)

// apiStatus contains the processor status for the API
type apiStatus struct {
	api.KeyValue

	p *Pool
}

// Update updates the processor status information
func (a *apiStatus) Update() error {
	tooOld, future := a.p.guard.Dropped()
	a.SetEntry("droppedTooOldEvents", api.Number(tooOld))
	a.SetEntry("droppedFutureEvents", api.Number(future))
	a.SetEntry("maximumEventAge", api.Number(int64(a.p.guard.MaximumAge()/time.Second)))
	a.SetEntry("maximumFutureEventAge", api.Number(int64(a.p.guard.MaximumFutureAge()/time.Second)))
	return nil
}
