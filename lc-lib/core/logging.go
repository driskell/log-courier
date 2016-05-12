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

package core

import (
	"io/ioutil"
	golog "log"
	"os"

	"gopkg.in/op/go-logging.v1"
)

var log *logging.Logger

func init() {
	log = logging.MustGetLogger("core")
}

type defaultLogBackend struct {
	file *os.File
	path string
}

func newDefaultLogBackend(path string, prefix string, flag int) (*defaultLogBackend, error) {
	ret := &defaultLogBackend{
		path: path,
	}

	golog.SetPrefix(prefix)
	golog.SetFlags(flag)

	err := ret.Reopen()
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (f *defaultLogBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	golog.Print(rec.Formatted(calldepth + 1))
	return nil
}

func (f *defaultLogBackend) Reopen() (err error) {
	var newFile *os.File

	newFile, err = os.OpenFile(f.path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0640)
	if err != nil {
		return
	}

	// Switch to new output before closing
	golog.SetOutput(newFile)

	if f.file != nil {
		f.file.Close()
	}

	f.file = newFile

	return nil
}

func (f *defaultLogBackend) Close() {
	// Discard logs before closing
	golog.SetOutput(ioutil.Discard)

	if f.file != nil {
		f.file.Close()
	}

	f.file = nil
}
