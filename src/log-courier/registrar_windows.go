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

package main

import (
  "encoding/json"
  "log"
  "os"
)

func (r *Registrar) WriteRegistry(state map[string]*FileState) {
  fname := r.persistdir + string(os.PathSeparator) + r.statefile
  tname := fname + ".new"
  file, err := os.Create(tname)
  if err != nil {
    log.Printf("Failed to open %s for writing: %s\n", tname, err)
    return
  }

  encoder := json.NewEncoder(file)
  encoder.Encode(state)
  file.Close()

  if _, err = os.Stat(fname); err != nil && os.IsNotExist(err) {
  } else {
    err = os.Remove(fname)
    if err != nil {
      log.Printf("Registrar save problem: Failed to delete previous file: %s\n", err)
    }
  }

  err = os.Rename(tname, fname)
  if err != nil {
    log.Printf("Registrar save problem: Failed to move the new file into place: %s\n", err)
  }
}
