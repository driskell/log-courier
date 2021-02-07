/*
 * Copyright 2015-2016 Jason Woods.
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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"unicode"
	"unicode/utf8"
)

const platformHeader = `// THIS IS A GO GENERATED FILE

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

`

// Generate platform.go
// It should contain platform specific defaults, such as the default
// configuration location and persist directory.
// Useful for package maintainers
func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: go run <path-to-lc-lib>/config/generate.go -- <target> <package-name> <configs>...")
	}

	targetFile := os.Args[1] + ".go"

	platformFile := platformHeader
	platformFile += fmt.Sprintf("package %s\n\n", os.Args[2])

	config := parseConfigArgs(os.Args[3:])
	platformFile += generatePackageImports(config)
	platformFile += generateInit(config)

	platformFileBytes := []byte(os.ExpandEnv(platformFile))
	if err := ioutil.WriteFile(targetFile, platformFileBytes, 0644); err != nil {
		log.Fatalf("Failed to write %s: %s", targetFile, err)
	}
}

func parseConfigArgs(args []string) map[string][]string {
	config := map[string][]string{}

	for _, v := range args {
		envSplit := strings.SplitN(v, ".", 2)
		if len(envSplit) != 2 {
			envSplit = []string{".", envSplit[0]}
		}

		config[envSplit[0]] = append(config[envSplit[0]], envSplit[1])
	}

	return config
}

func generatePackageImports(config map[string][]string) string {
	platformFile := "import (\n"
	for pkg := range config {
		if pkg == "." {
			continue
		}
		platformFile += fmt.Sprintf("\t\"github.com/driskell/log-courier/lc-lib/%s\"\n", pkg)
	}
	platformFile += ")\n\n"
	return platformFile
}

func generateInit(config map[string][]string) string {
	first := true
	platformFile := "func init() {\n"
	for pkg, nameArr := range config {
		for _, name := range nameArr {
			if !first {
				platformFile += "\n"
			} else {
				first = false
			}
			platformFile += generateInitSegment(pkg, name)
		}
	}
	platformFile += "}\n"
	return platformFile
}

func generateInitSegment(pkg, name string) string {
	var envName string

	nameSplit := strings.SplitN(name, ":", 2)
	if len(nameSplit) != 2 {
		envArr := split(name)
		for i := range envArr {
			envArr[i] = strings.ToUpper(envArr[i])
		}
		envName = "LC_" + strings.Join(envArr, "_")
	} else {
		envName = nameSplit[1]
	}

	platformFile := fmt.Sprintf("\t// %s\n", envName)
	if pkg == "." {
		platformFile += fmt.Sprintf("\t%s = \"${%s}\"\n", nameSplit[0], envName)
	} else {
		platformFile += fmt.Sprintf("\t%s.%s = \"${%s}\"\n", pkg, nameSplit[0], envName)
	}
	return platformFile
}

// https://github.com/fatih/camelcase/blob/9595d55ea95706cdcbe9ff3092ae8a8282aa7778/camelcase.go
//
// The MIT License (MIT)
// Copyright (c) 2015 Fatih Arslan
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
//
// Split splits the camelcase word and returns a list of words. It also
// supports digits. Both lower camel case and upper camel case are supported.
// For more info please check: http://en.wikipedia.org/wiki/CamelCase
//
// Examples
//
//   "" =>                     [""]
//   "lowercase" =>            ["lowercase"]
//   "Class" =>                ["Class"]
//   "MyClass" =>              ["My", "Class"]
//   "MyC" =>                  ["My", "C"]
//   "HTML" =>                 ["HTML"]
//   "PDFLoader" =>            ["PDF", "Loader"]
//   "AString" =>              ["A", "String"]
//   "SimpleXMLParser" =>      ["Simple", "XML", "Parser"]
//   "vimRPCPlugin" =>         ["vim", "RPC", "Plugin"]
//   "GL11Version" =>          ["GL", "11", "Version"]
//   "99Bottles" =>            ["99", "Bottles"]
//   "May5" =>                 ["May", "5"]
//   "BFG9000" =>              ["BFG", "9000"]
//   "BöseÜberraschung" =>     ["Böse", "Überraschung"]
//   "Two  spaces" =>          ["Two", "  ", "spaces"]
//   "BadUTF8\xe2\xe2\xa1" =>  ["BadUTF8\xe2\xe2\xa1"]
//
// Splitting rules
//
//  1) If string is not valid UTF-8, return it without splitting as
//     single item array.
//  2) Assign all unicode characters into one of 4 sets: lower case
//     letters, upper case letters, numbers, and all other characters.
//  3) Iterate through characters of string, introducing splits
//     between adjacent characters that belong to different sets.
//  4) Iterate through array of split strings, and if a given string
//     is upper case:
//       if subsequent string is lower case:
//         move last character of upper case string to beginning of
//         lower case string
func split(src string) (entries []string) {
	// don't split invalid utf8
	if !utf8.ValidString(src) {
		return []string{src}
	}
	entries = []string{}
	var runes [][]rune
	lastClass := 0
	class := 0
	// split into fields based on class of unicode character
	for _, r := range src {
		switch true {
		case unicode.IsLower(r):
			class = 1
		case unicode.IsUpper(r):
			class = 2
		case unicode.IsDigit(r):
			class = 3
		default:
			class = 4
		}
		if class == lastClass {
			runes[len(runes)-1] = append(runes[len(runes)-1], r)
		} else {
			runes = append(runes, []rune{r})
		}
		lastClass = class
	}
	// handle upper case -> lower case sequences, e.g.
	// "PDFL", "oader" -> "PDF", "Loader"
	for i := 0; i < len(runes)-1; i++ {
		if unicode.IsUpper(runes[i][0]) && unicode.IsLower(runes[i+1][0]) {
			runes[i+1] = append([]rune{runes[i][len(runes[i])-1]}, runes[i+1]...)
			runes[i] = runes[i][:len(runes[i])-1]
		}
	}
	// construct []string from results
	for _, s := range runes {
		if len(s) > 0 {
			entries = append(entries, string(s))
		}
	}
	return
}
