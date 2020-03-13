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

	"github.com/fatih/camelcase"
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
			log.Fatalf("Config variable needs package specifier: %s", v)
		}

		config[envSplit[0]] = append(config[envSplit[0]], envSplit[1])
	}

	return config
}

func generatePackageImports(config map[string][]string) string {
	platformFile := "import (\n"
	for pkg := range config {
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
		envArr := camelcase.Split(name)
		for i := range envArr {
			envArr[i] = strings.ToUpper(envArr[i])
		}
		envName = "LC_" + strings.Join(envArr, "_")
	} else {
		envName = nameSplit[1]
	}

	platformFile := fmt.Sprintf("\t// %s\n", envName)
	platformFile += fmt.Sprintf("\t%s.%s = \"${%s}\"\n", pkg, nameSplit[0], envName)
	return platformFile
}
