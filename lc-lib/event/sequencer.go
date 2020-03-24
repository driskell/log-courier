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

import "fmt"

// bundleMarkSequence type
type bundleMarkSequencer string

const (
	bundleMarkSequencerPos      bundleMarkSequencer = "p"
	bundleMarkSequencerEnforced bundleMarkSequencer = "e"
)

// Sequencer struct
type Sequencer struct {
	inputSequence  uint64
	outputSequence uint64
	pending        int
	delayed        []*Bundle
}

// NewSequencer creates a new sequencer
func NewSequencer() *Sequencer {
	return &Sequencer{}
}

// Track marks the given bundle and tracks its eventual ordering
// Enforce will store bundles until it has a valid sequence
// that matches the order in which Track was called against the bundles
// and then begin returning bundles
func (s *Sequencer) Track(bundle *Bundle) {
	if bundle.Value(bundleMarkSequencerPos) != nil {
		panic(fmt.Sprintf("Attempt to Track already tracked Bundle in Sequencer (pos: %d)", bundle.Value(bundleMarkSequencerPos).(uint64)))
	}
	bundle.Mark(bundleMarkSequencerPos, s.inputSequence)
	s.inputSequence++
	s.pending++
}

// Enforce will take the given bundle and return as follows:
//   If the bundle is out of order and we are missing a bundle, nil
//   will be returned
//   If the bundle is the next bundle in the sequence it is returned
//   along with any other bundles that appeared out of order, in the
//   correct order in the output slice
func (s *Sequencer) Enforce(bundle *Bundle) []*Bundle {
	if bundle.Value(bundleMarkSequencerPos) == nil {
		panic("Attempt to Enforce untracked Bundle in Sequencer")
	}
	if bundle.Value(bundleMarkSequencerEnforced) != nil {
		panic(fmt.Sprintf("Attempt to Enforce twice a tracked Bundle in Sequencer (pos: %d)", bundle.Value(bundleMarkSequencerPos).(uint64)))
	}
	bundle.Mark(bundleMarkSequencerEnforced, true)
	// Sequence check - is it in order?
	if bundle.Value(bundleMarkSequencerPos).(uint64) == s.outputSequence {
		s.outputSequence++
		s.pending--
		// NOTE: Optimised for correct ordering so works best when mostly in order
		result := []*Bundle{bundle}
		for {
			startSequence := s.outputSequence
			targetIdx := 0
			for idx, entry := range s.delayed {
				if entry.Value(bundleMarkSequencerPos).(uint64) == s.outputSequence {
					s.outputSequence++
					s.pending--
					result = append(result, entry)
				} else {
					if targetIdx != idx {
						s.delayed[targetIdx] = entry
					}
					targetIdx++
				}
			}
			for targetIdx := 0; targetIdx < len(s.delayed); targetIdx++ {
				// Free memory - but keep slice capacity
				s.delayed[targetIdx] = nil
			}
			s.delayed = s.delayed[:targetIdx]
			if startSequence == s.outputSequence {
				break
			}
		}
		return result
	}

	s.delayed = append(s.delayed, bundle)
	return nil
}

// Len returns the number of bundles tracked that have not yet being output
// from Enforce. This includes any that haven't been seen by Enforce yet
func (s *Sequencer) Len() int {
	return s.pending
}

// Delayed returns the number of bundles held delayed due to missing bundles
func (s *Sequencer) Delayed() int {
	return len(s.delayed)
}
