/*
 * Copyright 2014-2015 Jason Woods.
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

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGrokLoadFromReader(t *testing.T) {
	grok := NewGrok()
	err := grok.loadPatternsFromReader(strings.NewReader(`
ALL (%{SOME}*)
# This is a comment
SOME .
	`))
	if err != nil {
		t.Fatalf("Load from reader failed: %s", err)
		return
	}
	if len(grok.compiled) != 2 {
		t.Fatalf("Unexpected compiled patterns: %d", len(grok.compiled))
	}
	if grok.compiled["ALL"].pattern != "(.*)" {
		t.Fatalf("Unexpected pattern: %s", grok.compiled["ALL"].pattern)
	}
	if grok.compiled["SOME"].pattern != "." {
		t.Fatalf("Unexpected pattern: %s", grok.compiled["SOME"].pattern)
	}
}

func TestGrokLoadFromReaderInvalid(t *testing.T) {
	grok := NewGrok()
	err := grok.loadPatternsFromReader(strings.NewReader(`
ALL (%{SOME}*)
INVALID
SOME .
	`))
	if err == nil {
		t.Fatalf("Load from reader unexpected succeeded")
	}
}
func TestGrokAddPattern(t *testing.T) {
	grok := NewGrok()
	err := grok.AddPattern("ALL", "(.*)")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	if grok.compiled["ALL"].pattern != "(.*)" {
		t.Fatalf("Unexpected pattern: %s", grok.compiled["ALL"].pattern)
	}
	if len(grok.compiled["ALL"].types) != 0 {
		t.Fatalf("Unexpected types: %v", grok.compiled["ALL"].types)
	}
}

func TestGrokAddPatternReference(t *testing.T) {
	grok := NewGrok()
	err := grok.AddPattern("SOME", ".")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	err = grok.AddPattern("ALL", "(%{SOME}*)")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	if grok.compiled["ALL"].pattern != "(.*)" {
		t.Fatalf("Unexpected pattern: %s", grok.compiled["ALL"].pattern)
	}
	if len(grok.compiled["ALL"].types) != 0 {
		t.Fatalf("Unexpected types: %v", grok.compiled["ALL"].types)
	}
}

func TestGrokAddPatternNamedReference(t *testing.T) {
	grok := NewGrok()
	err := grok.AddPattern("SOME", ".")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	err = grok.AddPattern("ALL", "(%{SOME:some}*)")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	if grok.compiled["ALL"].pattern != "((?P<some>.)*)" {
		t.Fatalf("Unexpected pattern: %s", grok.compiled["ALL"].pattern)
	}
	if len(grok.compiled["ALL"].types) != 0 {
		t.Fatalf("Unexpected types: %v", grok.compiled["ALL"].types)
	}
}

func TestGrokAddPatternDelayed(t *testing.T) {
	grok := NewGrok()
	err := grok.AddPattern("ALL", "(%{SOME:some}*)")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	err = grok.AddPattern("SOME", ".")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	if grok.compiled["ALL"].pattern != "((?P<some>.)*)" {
		t.Fatalf("Unexpected pattern: %s", grok.compiled["ALL"].pattern)
	}
	if len(grok.compiled["ALL"].types) != 0 {
		t.Fatalf("Unexpected types: %v", grok.compiled["ALL"].types)
	}
}

func TestGrokAddPatternMultipleDelayed(t *testing.T) {
	grok := NewGrok()
	err := grok.AddPattern("ALL", "(%{SOME:some}*)%{SUFFIX}")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	err = grok.AddPattern("MOST", "(%{SOME:most}{52,})")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	err = grok.AddPattern("SOME", ".")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	err = grok.AddPattern("SUFFIX", "LogLevel:\\s+")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	if grok.compiled["ALL"].pattern != "((?P<some>.)*)LogLevel:\\s+" {
		t.Fatalf("Unexpected pattern: %s", grok.compiled["ALL"].pattern)
	}
	if grok.compiled["MOST"].pattern != "((?P<most>.){52,})" {
		t.Fatalf("Unexpected pattern: %s", grok.compiled["MOST"].pattern)
	}
	if len(grok.compiled["ALL"].types) != 0 {
		t.Fatalf("Unexpected types: %v", grok.compiled["ALL"].types)
	}
}

func TestGrokAddPatternDeeplyNested(t *testing.T) {
	grok := NewGrok()
	err := grok.AddPattern("ALL", "(%{MOST:some}*)")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	err = grok.AddPattern("MOST", "%{SOME}")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	err = grok.AddPattern("SOME", ".")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	if grok.compiled["ALL"].pattern != "((?P<some>.)*)" {
		t.Fatalf("Unexpected pattern: %s", grok.compiled["ALL"].pattern)
	}
	if len(grok.compiled["ALL"].types) != 0 {
		t.Fatalf("Unexpected types: %v", grok.compiled["ALL"].types)
	}
}

func TestGrokAddPatternIgnorePartial(t *testing.T) {
	grok := NewGrok()
	err := grok.AddPattern("ALL", "(%{INCOMPLETE.*)")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	if grok.compiled["ALL"].pattern != "(%{INCOMPLETE.*)" {
		t.Fatalf("Unexpected pattern: %s", grok.compiled["ALL"].pattern)
	}
	if len(grok.compiled["ALL"].types) != 0 {
		t.Fatalf("Unexpected types: %v", grok.compiled["ALL"].types)
	}
}

func TestGrokAddPatternTypes(t *testing.T) {
	grok := NewGrok()
	err := grok.AddPattern("ALL", "%{SOME:p:int}%{SOME:some:float}*")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	err = grok.AddPattern("SOME", ".")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	if grok.compiled["ALL"].pattern != "(?P<p>.)(?P<some>.)*" {
		t.Fatalf("Unexpected pattern: %s", grok.compiled["ALL"].pattern)
	}
	typesJSON, _ := json.Marshal(grok.compiled["ALL"].types)
	if string(typesJSON) != `{"p":"int","some":"float"}` {
		t.Fatalf("Unexpected types: %s", typesJSON)
	}
}
