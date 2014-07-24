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

type TransportRegistrar interface {
  NewFactory(string, string, map[string]interface{}) (TransportFactory, error)
}

type TransportFactory interface {
  NewTransport(*NetworkConfig) (Transport, error)
}

type Transport interface {
  ReloadConfig(*NetworkConfig) int
  Init() error
  CanSend() <-chan int
  Write(string, []byte) error
  Read() <-chan interface{}
  Shutdown()
}

var transportRegistry map[string]TransportRegistrar = make(map[string]TransportRegistrar)

func RegisterTransport(registrar TransportRegistrar, name string) {
  transportRegistry[name] = registrar
}

func AvailableTransports() (ret []string) {
  ret = make([]string, 0, len(transportRegistry))
  for k := range transportRegistry {
    ret = append(ret, k)
  }
  return
}

func NewTransportFactory(config_path string, name string, config map[string]interface{}) (TransportFactory, error) {
  if registrar, ok := transportRegistry[name]; ok {
    return registrar.NewFactory(name, config_path, config)
  }
  return nil, nil
}
