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
  "flag"
  "fmt"
  "lc-lib/admin"
)

func main() {
  var host string
  var port int

  flag.StringVar(&host, "host", "127.0.0.1", "the Log Courier host to connect to (default 127.0.0.1)")
  flag.IntVar(&port, "port", 1234, "the Log Courier monitor port (default 1234)")

  flag.Parse()

  admin, err := admin.NewClient(host, port)
  if err != nil {
    fmt.Printf("Failed to connect to Log Courier: %s", err)
  }

  snapshots := admin.FetchSnapshot()

  for _, snap := range snapshots {
    fmt.Printf("%s", snap.Description())
    for i, j := 0, snap.NumEntries(); i < j; i = i+1 {
      k, v := snap.Entry(i)
      fmt.Printf("  %s = %s", k, v)
    }
  }
}
