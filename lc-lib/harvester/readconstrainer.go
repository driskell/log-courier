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

package harvester

import (
	"io"
)

// readConstrainer provides an io.Reader that has a maximum limit
// This is used by other readers that need to keep memory under control
type readConstrainer struct {
	rd      io.Reader
	maxRead int
}

// newReadConstrainer creates a new readConstrainer with the given reader and
// maximum read size
func newReadConstrainer(rd io.Reader, maxRead int) *readConstrainer {
	return &readConstrainer{
		rd:      rd,
		maxRead: maxRead,
	}
}

// setMaxRead sets the new read limit and returned what is left of the old limt
func (r *readConstrainer) setMaxRead(maxRead int) int {
	oldMaxRead := r.maxRead
	r.maxRead = maxRead
	return oldMaxRead
}

// Read reads data from the reader up until the maximum read size has been
// reached, at which point it returns an error
func (r *readConstrainer) Read(b []byte) (int, error) {
	if r.maxRead == 0 {
		return 0, ErrMaxDataSizeExceeded
	}

	var n int
	var err error
	if cap(b) < r.maxRead {
		n, err = r.rd.Read(b)
	} else {
		n, err = r.rd.Read(b[:r.maxRead])
	}
	r.maxRead -= n
	return n, err
}
