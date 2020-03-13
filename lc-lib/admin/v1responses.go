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

package admin

import (
	"encoding/gob"
	"time"

	"github.com/driskell/log-courier/lc-lib/core"
)

type Response struct {
	Response interface{}
}

type PongResponse struct {
}

type ReloadResponse struct {
}

type ErrorResponse struct {
	Message string
}

func (e *ErrorResponse) Error() string {
	return e.Message
}

func init() {
	// Response structure
	gob.Register(&Response{})

	// General error
	gob.Register(&ErrorResponse{})

	// PONG
	gob.Register(&PongResponse{})

	// RELD
	gob.Register(&ReloadResponse{})

	// SNAP
	gob.Register(&core.Snapshot{})
	// SNAP - time.Time
	gob.Register(time.Now())
	// SNAP - time.Duration
	gob.Register(time.Since(time.Now()))
}
