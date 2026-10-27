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
	"strconv"
	"sync"
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

// apiRoutines lists each processor routine for the API
type apiRoutines struct {
	api.Array

	p     *Pool
	mutex sync.Mutex
	count int
}

// Update lists the routines currently running, then updates each of them
func (a *apiRoutines) Update() error {
	a.mutex.Lock()
	routines := int(a.p.routines.Load())
	for ; a.count < routines; a.count++ {
		a.AddEntry(strconv.Itoa(a.count), &apiRoutine{p: a.p, id: a.count})
	}
	for ; a.count > routines; a.count-- {
		a.RemoveEntry(strconv.Itoa(a.count - 1))
	}
	a.mutex.Unlock()

	return a.Array.Update()
}

// apiRoutine contains a processor routine's status for the API
type apiRoutine struct {
	api.KeyValue

	p  *Pool
	id int
}

// Update updates the routine status information
func (a *apiRoutine) Update() error {
	a.SetEntry("id", api.Number(a.id))
	a.SetEntry("processedEvents", api.Number(a.p.processed[a.id].Load()))
	a.SetEntry("speed", api.Float(a.p.speed(a.id)))
	return nil
}
