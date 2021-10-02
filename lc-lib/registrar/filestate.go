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
)

// FileInfo holds information about a file that has been found on disk
// Each time we scan we can use it to check whether a given file on disk is the file this structure holds information on
type FileInfo struct {
	fileinfo os.FileInfo
}

// NewFileInfo returns a new FileInfo that can be compared to other instances to see if they are the same file on disk
func NewFileInfo(fileinfo os.FileInfo) *FileInfo {
	return &FileInfo{
		fileinfo: fileinfo,
	}
}

// SameAs returns true if this file info is the same as the other
func (fs *FileInfo) SameAs(info os.FileInfo) bool {
	return os.SameFile(info, fs.fileinfo)
}

// Stat returns the stdlib FileInfo for this entry
func (fs *FileInfo) Stat() os.FileInfo {
	return fs.fileinfo
}

// Update stores updated file information for this file
func (fs *FileInfo) Update(fileinfo os.FileInfo, identity *FileIdentity) {
	fs.fileinfo = fileinfo
}

// FileState holds persisted progress information about a file that has yet to be discovered on disk
// It is loaded from the registrar state and holds operating system specific identifiers
// It can be given a file on disk and will return, fairly reliably, if it represents the same file, using
// operating system specific comparison of file identifiers
// It should be replaced by a FileInfo as soon as the file is located as comparing real OS structures is
// more reliable than using the identifiers
type FileState struct {
	FileStateOS
	Source *string `json:"source,omitempty"`
	Offset int64   `json:"offset,omitempty"`
}

// Stat returns nil for a yet to be discovered file
func (fs *FileState) Stat() os.FileInfo {
	return nil
}

// Update replaces a FileState with a FileInfo that references a now found file on disk
func (fs *FileState) Update(fileinfo os.FileInfo, identity *FileIdentity) {
	// Promote to a FileInfo
	(*identity) = NewFileInfo(fileinfo)
}

// FileIdentity is the interface that FileInfo and FileState implement
type FileIdentity interface {
	SameAs(os.FileInfo) bool
	Stat() os.FileInfo
	Update(os.FileInfo, *FileIdentity)
}
