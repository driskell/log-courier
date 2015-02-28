/*
 * Copyright 2014 Jason Woods.
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

type FileState struct {
	FileStateOS
	Source *string `json:"source,omitempty"`
	Offset int64   `json:"offset,omitempty"`
}

type FileInfo struct {
	fileinfo os.FileInfo
}

func NewFileInfo(fileinfo os.FileInfo) *FileInfo {
	return &FileInfo{
		fileinfo: fileinfo,
	}
}

func (fs *FileInfo) SameAs(info os.FileInfo) bool {
	return os.SameFile(info, fs.fileinfo)
}

func (fs *FileInfo) Stat() os.FileInfo {
	return fs.fileinfo
}

func (fs *FileInfo) Update(fileinfo os.FileInfo, identity *FileIdentity) {
	fs.fileinfo = fileinfo
}

func (fs *FileState) Stat() os.FileInfo {
	return nil
}

func (fs *FileState) Update(fileinfo os.FileInfo, identity *FileIdentity) {
	// Promote to a FileInfo
	(*identity) = NewFileInfo(fileinfo)
}

type FileIdentity interface {
	SameAs(os.FileInfo) bool
	Stat() os.FileInfo
	Update(os.FileInfo, *FileIdentity)
}
