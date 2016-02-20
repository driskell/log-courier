/*
* Copyright 2014 Jason Woods.
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

package main

import (
	"gopkg.in/op/go-logging.v1"
	"io/ioutil"
	golog "log"
	"os"
)

var log *logging.Logger

func init() {
	log = logging.MustGetLogger("log-courier")
}

type DefaultLogBackend struct {
	file *os.File
	path string
}

func NewDefaultLogBackend(path string, prefix string, flag int) (*DefaultLogBackend, error) {
	ret := &DefaultLogBackend{
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

func (f *DefaultLogBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	golog.Print(rec.Formatted(calldepth + 1))
	return nil
}

func (f *DefaultLogBackend) Reopen() (err error) {
	var new_file *os.File

	new_file, err = os.OpenFile(f.path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0640)
	if err != nil {
		return
	}

	// Switch to new output before closing
	golog.SetOutput(new_file)

	if f.file != nil {
		f.file.Close()
	}

	f.file = new_file

	return nil
}

func (f *DefaultLogBackend) Close() {
	// Discard logs before closing
	golog.SetOutput(ioutil.Discard)

	if f.file != nil {
		f.file.Close()
	}

	f.file = nil
}
