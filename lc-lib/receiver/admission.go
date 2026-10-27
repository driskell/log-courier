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

package receiver

import (
	"math"

	"github.com/driskell/log-courier/lc-lib/transports"
)

// admissionQuantum is the byte credit a waiting connection earns each time admission reaches it
const admissionQuantum = 64 * 1024

// admissionRequest is a connection waiting for the receiver pool to take the events held in its receive window
type admissionRequest struct {
	connection interface{}
	window     *transports.ReceiveWindow
}

// admissionControl decides when the events each connection holds in its receive window are taken into the queue.
// Connections with nothing queued are admitted first, so idle connections are not held back behind busy ones. Others
// are then admitted round robin while the queue is under budget and their byte credit is not negative, so a
// connection sending more data waits longer between admissions.
type admissionControl struct {
	waiting []*admissionRequest
	// credit is each connection's byte credit, which goes negative when charged and is earned back while waiting
	credit map[interface{}]int
}

func newAdmissionControl() *admissionControl {
	return &admissionControl{
		credit: make(map[interface{}]int),
	}
}

// request adds a connection to those waiting for their held events to be taken
func (a *admissionControl) request(connection interface{}, window *transports.ReceiveWindow) {
	a.waiting = append(a.waiting, &admissionRequest{connection: connection, window: window})
}

// charge records the size of the events taken from a connection
func (a *admissionControl) charge(connection interface{}, size int) {
	a.credit[connection] -= size
}

// forget removes a connection that has gone
func (a *admissionControl) forget(connection interface{}) {
	kept := a.waiting[:0]
	for _, waiting := range a.waiting {
		if waiting.connection != connection {
			kept = append(kept, waiting)
		}
	}
	a.waiting = kept
	delete(a.credit, connection)
}

// admit grants the requests that may proceed, given whether a connection has anything queued and whether the queue
// is under budget, calling grant for each
func (a *admissionControl) admit(queued func(connection interface{}) bool, room func() bool, grant func(*admissionRequest)) {
	kept := a.waiting[:0]
	for _, waiting := range a.waiting {
		if !queued(waiting.connection) {
			// Debt is forgiven once nothing is queued, so a connection cannot build debt while the queue is idle
			a.credit[waiting.connection] = 0
			grant(waiting)
		} else {
			kept = append(kept, waiting)
		}
	}
	a.waiting = kept

	unserved, minRounds := 0, math.MaxInt
	for len(a.waiting) != 0 && room() {
		waiting := a.waiting[0]
		a.waiting = a.waiting[1:]
		// Credit is earned on each visit but capped at zero, so a connection cannot bank credit while it has none to spend
		credit := min(0, a.credit[waiting.connection]+admissionQuantum)
		a.credit[waiting.connection] = credit
		if credit >= 0 {
			grant(waiting)
			unserved, minRounds = 0, math.MaxInt
			continue
		}

		a.waiting = append(a.waiting, waiting)
		minRounds = min(minRounds, (-credit+admissionQuantum-1)/admissionQuantum)
		unserved++
		if unserved == len(a.waiting) {
			// Nobody could be admitted in a full round, so credit the rounds needed until someone has credit again
			for _, each := range a.waiting {
				a.credit[each.connection] = min(0, a.credit[each.connection]+(minRounds-1)*admissionQuantum)
			}
			unserved, minRounds = 0, math.MaxInt
		}
	}
}
