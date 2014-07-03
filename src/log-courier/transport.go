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

type TransportFactory interface {
  NewTransport(*NetworkConfig) (Transport, error)
}

type Transport interface {
  ReloadConfig(*NetworkConfig) int
  Connect() error
  CanSend() <-chan int
  Write(string, []byte) error
  Read() <-chan interface{}
  Disconnect()
}
