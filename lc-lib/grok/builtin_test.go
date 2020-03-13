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

package grok

import "testing"

func TestDefaultPatterns(t *testing.T) {
	grok := NewGrok(false)
	for name, pattern := range DefaultPatterns {
		err := grok.AddPattern(name, pattern)
		if err != nil {
			t.Fatalf("Unexpected default pattern add failure: %s", err)
		}
	}
	for name, pattern := range DefaultPatterns {
		_, err := grok.CompilePattern(pattern, nil)
		if err != nil {
			t.Fatalf("Unexpected default pattern compile failure for %s: %s", name, err)
		}
	}
}
