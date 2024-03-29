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

package publisher

import (
	"github.com/driskell/log-courier/lc-lib/admin/api"
)

type apiStatus struct {
	api.KeyValue

	p *Publisher
}

// Update updates the publisher status information
func (a *apiStatus) Update() error {
	// Update the values and pass through to node
	a.p.mutex.RLock()
	a.SetEntry("speed", api.Float(a.p.lineSpeed))
	a.SetEntry("publishedLines", api.Number(a.p.lastLineCount))
	a.SetEntry("pendingPayloads", api.Number(a.p.numPayloads))
	a.SetEntry("maxPendingPayloads", api.Number(a.p.netConfig.MaxPendingPayloads))
	a.p.mutex.RUnlock()

	return nil
}
