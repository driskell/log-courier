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

func WriteRegistry(state map[string]*FileState, path string) {
  tmp := path + ".new"
  file, err := os.Create(tmp)
  if err != nil {
    log.Printf("Failed to open .logstash-forwarder.new for writing: %s\n", err)
    return
  }

  encoder := json.NewEncoder(file)
  encoder.Encode(state)
  file.Close()

  old := path + ".old"

  if _, err = os.Stat(old); err != nil && os.IsNotExist(err) {
  } else {
    err = os.Remove(old)
    if err != nil {
      log.Printf("Registrar save problem: Failed to delete backup file: %s\n", err)
    }
  }

  if _, err = os.Stat(path); err != nil && os.IsNotExist(err) {
  } else {
    err = os.Rename(path, old)
    if err != nil {
      log.Printf("Registrar save problem: Failed to perform backup: %s\n", err)
    }
  }

  err = os.Rename(tmp, path)
  if err != nil {
    log.Printf("Registrar save problem: Failed to move the new file into place: %s\n", err)
  }
}
