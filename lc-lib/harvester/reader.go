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
	"errors"
)

var (
	// ErrMaxDataSizeTruncation is returned when the read data was longer than the
	// maximum allowed size and had to be truncated
	ErrMaxDataSizeTruncation = errors.New("data exceeded \"max line bytes\" and was truncated")

	// ErrMaxLineBytesExceeded is reported when max line bytes is exceeded in an unrecoverable way
	ErrMaxDataSizeExceeded = errors.New("data exceeds \"max line bytes\" and prevents the file from being processed")
)

// Reader is implemented by the various harvester readers and reads events from
// a file or stream
type Reader interface {
	BufferedLen() int
	ReadItem() (map[string]interface{}, int, error)
	Reset()
}
