/*
 * Copyright 2012-2020 Jason Woods and contributors
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

package event

import (
	"testing"
)

func TestSequencerConsecutive(t *testing.T) {
	s := NewSequencer()
	var bundles [5]*Bundle
	bundles[0] = NewBundle(nil)
	bundles[1] = NewBundle(nil)
	bundles[2] = NewBundle(nil)
	bundles[3] = NewBundle(nil)
	bundles[4] = NewBundle(nil)
	for _, bundle := range bundles {
		s.Track(bundle)
	}
	if s.Len() != 5 {
		t.Fatal("Unexpected bundle pending count")
	}
	result := []*Bundle{}
	for _, bundle := range bundles {
		result = append(result, s.Enforce(bundle)...)
		if s.Delayed() != 0 {
			t.Fatal("Unexpected delayed count")
		}
	}
	if len(result) != 5 {
		t.Fatalf("Unexpected bundle count: %d", len(result))
	}
	for idx, entry := range result {
		if entry != bundles[idx] {
			t.Fatal("Unexpected final bundle ordering")
		}
	}
}

func TestSequencerReversed(t *testing.T) {
	s := NewSequencer()
	var bundles [5]*Bundle
	bundles[0] = NewBundle(nil)
	bundles[1] = NewBundle(nil)
	bundles[2] = NewBundle(nil)
	bundles[3] = NewBundle(nil)
	bundles[4] = NewBundle(nil)
	for _, bundle := range bundles {
		s.Track(bundle)
	}
	result := []*Bundle{}
	for idx := 4; idx >= 0; idx-- {
		ret := s.Enforce(bundles[idx])
		if idx > 0 {
			if len(ret) != 0 {
				t.Fatal("Unexpected early bundle return")
			}
			if s.Delayed() != 5-idx {
				t.Fatal("Unexpected length return")
			}
		}
		result = append(result, ret...)
	}
	if len(result) != 5 {
		t.Fatalf("Unexpected bundle count: %d", len(result))
	}
	for idx, entry := range result {
		if entry != bundles[idx] {
			t.Fatal("Unexpected final bundle ordering")
		}
	}
}

func TestSequencerDoubleMark(t *testing.T) {
	defer func() {
		recover()
	}()
	s := NewSequencer()
	bundle := NewBundle(nil)
	s.Track(bundle)
	s.Track(bundle)
	t.Fatal("Double mark did not panic")
}

func TestSequencerDoubleEnforce(t *testing.T) {
	defer func() {
		recover()
	}()
	s := NewSequencer()
	bundle := NewBundle(nil)
	s.Track(bundle)
	s.Enforce(bundle)
	s.Enforce(bundle)
	t.Fatal("Double enforce did not panic")
}

func TestSequencerEnforceWithoutTrack(t *testing.T) {
	defer func() {
		recover()
	}()
	s := NewSequencer()
	bundle := NewBundle(nil)
	s.Enforce(bundle)
	t.Fatal("Enforce without track did not panic")
}
