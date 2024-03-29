/*
 * Copyright 2012-2020 Jason Woods and contributors
 *
 * This file is a modification of code from Logstash Forwarder.
 * Copyright 2012-2013 Jordan Sissel and contributors.
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
	"syscall"
)

// FileStateOS holds operating system specific identifiers for a file
type FileStateOS struct {
	Inode  uint32 `json:"inode,omitempty"`
	Device uint32 `json:"device,omitempty"`
}

// PopulateFileIds grabs the OS specific identifiers for the given file on disk
func (fs *FileStateOS) PopulateFileIds(info os.FileInfo) {
	fstat := info.Sys().(*syscall.Stat_t)
	fs.Inode = fstat.Ino
	fs.Device = fstat.Dev
}

// SameAs compares the given file using the stored identifiers
func (fs *FileStateOS) SameAs(info os.FileInfo) bool {
	state := &FileStateOS{}
	state.PopulateFileIds(info)
	return (fs.Inode == state.Inode && fs.Device == state.Device)
}
