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

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGrokLoadFromReader(t *testing.T) {
	grok := NewGrok(false)
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

func TestGrokLoadDefaults(t *testing.T) {
	grok := NewGrok(true)
	if len(grok.compiled) != len(DefaultPatterns) {
		t.Fatalf("Unexpected compiled default patterns: %d", len(grok.compiled))
	}
}

func TestGrokLoadFromReaderInvalid(t *testing.T) {
	grok := NewGrok(false)
	err := grok.loadPatternsFromReader(strings.NewReader(`
ALL (%{SOME}*)
INVALID
SOME .
	`))
	if err == nil {
		t.Fatalf("Load from reader unexpected succeeded")
	}
}

func TestGrokLoadFromReaderInvalidTypes(t *testing.T) {
	grok := NewGrok(false)
	err := grok.loadPatternsFromReader(strings.NewReader(`
ALL (%{SOME:name:invalid}*)
SOME .
	`))
	if err == nil {
		t.Fatalf("Load from reader unexpected succeeded")
	}
}

func TestGrokAddPattern(t *testing.T) {
	grok := NewGrok(false)
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
	grok := NewGrok(false)
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
	grok := NewGrok(false)
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
	grok := NewGrok(false)
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
	grok := NewGrok(false)
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
	grok := NewGrok(false)
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
	grok := NewGrok(false)
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
	grok := NewGrok(false)
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

func TestGrokAddPatternInvalidType(t *testing.T) {
	grok := NewGrok(false)
	err := grok.AddPattern("SOME", ".")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	err = grok.AddPattern("ALL", "%{SOME:p:invalid}%{SOME:some:float}*")
	if err == nil {
		t.Fatalf("Unexpected success: %s", grok.compiled["ALL"].pattern)
	}
}

func TestGrokAddPatternInvalidTypeDelayed(t *testing.T) {
	grok := NewGrok(false)
	err := grok.AddPattern("ALL", "%{MOST:p:invalid}%{SOME:some:float}*")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	err = grok.AddPattern("MOST", "%{SOME:p:int}%{SOME:some:float}*")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	err = grok.AddPattern("SOME", ".")
	if err == nil {
		t.Fatalf("Unexpected success: %s", grok.compiled["ALL"].pattern)
	}
}

func TestGrokMissing(t *testing.T) {
	grok := NewGrok(false)
	err := grok.AddPattern("ALL", "(%{SOME}*)")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	missingJSON, _ := json.Marshal(grok.MissingPatterns())
	if string(missingJSON) != `["SOME"]` {
		t.Fatalf("Unexpected missing pattern list: %s", missingJSON)
	}
}

func TestGrokCompilePattern(t *testing.T) {
	grok := NewGrok(false)
	pattern, err := grok.CompilePattern("(.*)", nil)
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	if pattern.String() != "(.*)" {
		t.Fatalf("Unexpected pattern: %s", pattern.String())
	}
	compiledPattern, _ := pattern.(*compiledPattern)
	if compiledPattern.re == nil {
		t.Fatal("Compiled pattern was not init")
	}
}

func TestGrokCompilePatternReference(t *testing.T) {
	grok := NewGrok(false)
	err := grok.AddPattern("ALL", "(.*)")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	pattern, err := grok.CompilePattern("%{ALL}", nil)
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	if pattern.String() != "(.*)" {
		t.Fatalf("Unexpected pattern: %s", pattern.String())
	}
}

func TestGrokCompilePatternFailed(t *testing.T) {
	grok := NewGrok(false)
	pattern, err := grok.CompilePattern("*", nil)
	if err == nil {
		t.Fatalf("Unexpected success: %s", pattern.String())
	}
}

func TestGrokCompilePatternFailedType(t *testing.T) {
	grok := NewGrok(false)
	err := grok.AddPattern("ALL", "(.*)")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	pattern, err := grok.CompilePattern("%{ALL:all:invalid}", nil)
	if err == nil {
		t.Fatalf("Unexpected success: %s", pattern.String())
	}
}

func TestGrokCompilePatternLocal(t *testing.T) {
	grok := NewGrok(false)
	err := grok.AddPattern("ALL", "(.*)")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	pattern, err := grok.CompilePattern("%{SOME}", map[string]string{
		"SOME": "%{ALL}",
	})
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	if pattern.String() != "(.*)" {
		t.Fatalf("Unexpected pattern: %s", pattern.String())
	}
	if len(grok.compiled) != 1 {
		t.Fatalf("Unexpected saved patterns: %v", grok.compiled)
	}
}

func TestGrokCompilePatternLocalNested(t *testing.T) {
	grok := NewGrok(false)
	pattern, err := grok.CompilePattern("%{SOME}*", map[string]string{
		"SOME": ".",
	})
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	if pattern.String() != ".*" {
		t.Fatalf("Unexpected pattern: %s", pattern.String())
	}
	if len(grok.compiled) != 0 {
		t.Fatalf("Unexpected saved patterns: %v", grok.compiled)
	}
}

func TestGrokCompilePatternLocalNestedDeep(t *testing.T) {
	grok := NewGrok(false)
	pattern, err := grok.CompilePattern("%{MORE}*", map[string]string{
		"MORE":  "%{SOME}%{OTHER}",
		"SOME":  "%{LAST}",
		"OTHER": "o",
		"LAST":  "t",
	})
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	if pattern.String() != "to*" {
		t.Fatalf("Unexpected pattern: %s", pattern.String())
	}
	if len(grok.compiled) != 0 {
		t.Fatalf("Unexpected saved patterns: %v", grok.compiled)
	}
}

func TestGrokCompilePatternLocalFailureNested(t *testing.T) {
	grok := NewGrok(false)
	err := grok.AddPattern("ALL", "(.*)")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	pattern, err := grok.CompilePattern("%{INVALID}", map[string]string{
		"INVALID": "%{ALL:p:invalid}",
	})
	if err == nil {
		t.Fatalf("Unexpected success: %s", pattern.String())
	}
}

func TestGrokCompilePatternLocalFailureNestedShallow(t *testing.T) {
	grok := NewGrok(false)
	err := grok.AddPattern("ALL", "(.*)")
	if err != nil {
		t.Fatalf("Unexpected failure: %s", err)
	}
	pattern, err := grok.CompilePattern("%{INVALID}", map[string]string{
		"INVALID": "%{WAIT}%{ALL:p:invalid}",
		"WAIT":    ".",
	})
	if err == nil {
		t.Fatalf("Unexpected success: %s", pattern.String())
	}
}

func TestGrokCompilePatternLocalFailureNestedDeep(t *testing.T) {
	grok := NewGrok(false)
	pattern, err := grok.CompilePattern("%{MORE}*", map[string]string{
		"MORE":  "%{SOME}%{OTHER:p:invalid}",
		"SOME":  "%{LAST}",
		"OTHER": "o",
		"LAST":  "t",
	})
	if err == nil {
		t.Fatalf("Unexpected success: %s", pattern.String())
	}
	if len(grok.compiled) != 0 {
		t.Fatalf("Unexpected saved patterns: %v", grok.compiled)
	}
}

func TestGrokCompilePatternLocalFailureMissing(t *testing.T) {
	grok := NewGrok(false)
	pattern, err := grok.CompilePattern("%{SOME}*", map[string]string{
		"SOME": "%{MISSING}",
	})
	if err == nil {
		t.Fatalf("Unexpected success: %s", pattern.String())
	}
}
