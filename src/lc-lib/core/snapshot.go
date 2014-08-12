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

type Snapshot struct {
  Desc    string
  Entries map[string]interface{}
  Keys    []string
  Subs    []*Snapshot
}

func NewSnapshot(desc string) *Snapshot {
  return &Snapshot{
    Desc:    desc,
    Entries: make(map[string]interface{}),
    Keys:    make([]string, 0),
    Subs:    make([]*Snapshot, 0),
  }
}

func (s *Snapshot) Description() string {
  return s.Desc
}

func (s *Snapshot) AddEntry(name string, value interface{}) {
  s.Entries[name] = value
  s.Keys = append(s.Keys, name)
}

func (s *Snapshot) EntryByName(name string) (interface{}, bool) {
  if v, ok := s.Entries[name]; ok {
    return v, true
  }

  return nil, false
}

func (s *Snapshot) Entry(i int) (string, interface{}) {
  if i < 0 || i >= len(s.Keys) {
    panic("Out of bounds")
  }

  return s.Keys[i], s.Entries[s.Keys[i]]
}

func (s *Snapshot) NumEntries() int {
  return len(s.Keys)
}

func (s *Snapshot) AddSub(sub *Snapshot) {
  s.Subs = append(s.Subs, sub)
}

func (s *Snapshot) Sub(i int) *Snapshot {
  if i < 0 || i >= len(s.Subs) {
    panic("Out of bounds")
  }

  return s.Subs[i]
}

func (s *Snapshot) NumSubs() int {
  return len(s.Subs)
}
