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

package courier

import (
	"compress/zlib"
	"io"
	"sync"
)

// decompressors holds zlib readers for reuse, as each holds a large decompression window
var decompressors sync.Pool

// newDecompressor returns a zlib reader for the given source, reusing a released one where possible
func newDecompressor(source io.Reader) (io.ReadCloser, error) {
	if decompressor, ok := decompressors.Get().(io.ReadCloser); ok {
		if err := decompressor.(zlib.Resetter).Reset(source, nil); err != nil {
			decompressors.Put(decompressor)
			return nil, err
		}
		return decompressor, nil
	}
	return zlib.NewReader(source)
}

// releaseDecompressor returns a zlib reader for reuse once nothing more will be read from it
func releaseDecompressor(decompressor io.ReadCloser) {
	decompressors.Put(decompressor)
}
