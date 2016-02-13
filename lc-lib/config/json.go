/*
 * Copyright 2014-2015 Jason Woods.
 *
 * This file is a modification of code from Logstash Forwarder.
 * Copyright 2012-2013 Jordan Sissel and contributors.
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

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// loadJSONFile loads the given JSON format file, stripping out our custom
// comments syntax before it does so
func (c *Config) loadJSONFile(path string, rawConfig interface{}) (err error) {
	stripped := new(bytes.Buffer)

	file, err := os.Open(path)
	if err != nil {
		err = fmt.Errorf("Failed to open config file: %s", err)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		err = fmt.Errorf("Stat failed for config file: %s", err)
		return
	}
	if stat.Size() == 0 {
		err = fmt.Errorf("Empty configuration file")
		return
	}
	if stat.Size() > (10 << 20) {
		err = fmt.Errorf("Config file too large (%s)", stat.Size())
		return
	}

	// Strip comments and read config into stripped
	var s, p, state int
	{
		// Pull the config file into memory
		buffer := make([]byte, stat.Size())
		_, err = file.Read(buffer)
		if err != nil {
			return
		}

		for p < len(buffer) {
			b := buffer[p]
			if state == 0 {
				// Main body
				if b == '"' {
					state = 1
				} else if b == '\'' {
					state = 2
				} else if b == '#' {
					state = 3
					stripped.Write(buffer[s:p])
				} else if b == '/' {
					state = 4
				}
			} else if state == 1 {
				// Double-quoted string
				if b == '\\' {
					state = 5
				} else if b == '"' {
					state = 0
				}
			} else if state == 2 {
				// Single-quoted string
				if b == '\\' {
					state = 6
				} else if b == '\'' {
					state = 0
				}
			} else if state == 3 {
				// End of line comment (#)
				if b == '\r' || b == '\n' {
					state = 0
					s = p + 1
				}
			} else if state == 4 {
				// Potential start of multiline comment
				if b == '*' {
					state = 7
					stripped.Write(buffer[s : p-1])
				} else {
					state = 0
				}
			} else if state == 5 {
				// Escape within double quote
				state = 1
			} else if state == 6 {
				// Escape within single quote
				state = 2
			} else if state == 7 {
				// Multiline comment (/**/)
				if b == '*' {
					state = 8
				}
			} else { // state == 8
				// Potential end of multiline comment
				if b == '/' {
					state = 0
					s = p + 1
				} else {
					state = 7
				}
			}
			p++
		}
		stripped.Write(buffer[s:p])
	}

	if stripped.Len() == 0 {
		err = fmt.Errorf("Empty configuration file")
		return
	}

	// Pull the entire structure into rawConfig
	if err = json.Unmarshal(stripped.Bytes(), rawConfig); err != nil {
		err = c.parseJSONSyntaxError(stripped.Bytes(), err)
		return
	}

	return
}

// parseSyntaxError parses a JSON Unmarshal error into a pretty error message
// when given the original JSON data and the received error
func (c *Config) parseJSONSyntaxError(js []byte, err error) error {
	jsonErr, ok := err.(*json.SyntaxError)
	if !ok {
		return err
	}

	start := bytes.LastIndex(js[:jsonErr.Offset], []byte("\n")) + 1
	end := bytes.Index(js[start:], []byte("\n"))
	if end >= 0 {
		end += start
	} else {
		end = len(js)
	}

	line, pos := bytes.Count(js[:start], []byte("\n")), int(jsonErr.Offset)-start-1

	var posStr string
	if pos > 0 {
		posStr = strings.Repeat(" ", pos)
	} else {
		posStr = ""
	}

	return fmt.Errorf("json: %s on line %d\n%s\n%s^", err, line, js[start:end], posStr)
}
