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

package registrar

import (
	"os"

	"github.com/driskell/log-courier/lc-lib/core"
)

type DiscoverEvent struct {
	stream   core.Stream
	source   string
	offset   int64
	fileinfo os.FileInfo
}

func NewDiscoverEvent(stream core.Stream, source string, offset int64, fileinfo os.FileInfo) *DiscoverEvent {
	return &DiscoverEvent{
		stream:   stream,
		source:   source,
		offset:   offset,
		fileinfo: fileinfo,
	}
}

func (e *DiscoverEvent) Process(state map[core.Stream]*FileState) {
	log.Debug("Registrar received a new file event for %s", e.source)

	// A new file we need to save offset information for so we can resume
	state[e.stream] = &FileState{
		Source: &e.source,
		Offset: e.offset,
	}
	state[e.stream].PopulateFileIds(e.fileinfo)
}
