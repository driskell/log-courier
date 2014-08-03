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

package core

type Transport interface {
  ReloadConfig(*NetworkConfig) int
  Init() error
  CanSend() <-chan int
  Write(string, []byte) error
  Read() <-chan interface{}
  Shutdown()
}

type TransportFactory interface {
  NewTransport(*NetworkConfig) (Transport, error)
}

type TransportRegistrar interface {
  NewFactory(*Config, string, string, map[string]interface{}) (TransportFactory, error)
}

var registered_Transports map[string]TransportRegistrar = make(map[string]TransportRegistrar)

func RegisterTransport(transport string, registrar TransportRegistrar) {
  registered_Transports[transport] = registrar
}

func AvailableTransports() (ret []string) {
  ret = make([]string, 0, len(registered_Transports))
  for k := range registered_Transports {
    ret = append(ret, k)
  }
  return
}
