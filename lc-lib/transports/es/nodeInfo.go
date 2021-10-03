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

package es

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// NodeInfo represents cluster information from Elasticsearch
type nodeInfo struct {
	Nodes map[string]*nodeInfoNode `json:"nodes"`
}

// MaxMajorVersion returns the maximum major version from the node list
func (n *nodeInfo) MaxMajorVersion() (int, error) {
	if len(n.Nodes) == 0 {
		return 0, errors.New("no nodes were found")
	}

	maxMajorVersion := 0
	for _, node := range n.Nodes {
		dotPos := strings.IndexRune(node.Version, '.')
		if dotPos == -1 {
			dotPos = len(node.Version)
		}
		majorVersion, err := strconv.Atoi(node.Version[0:dotPos])
		if err != nil {
			return 0, fmt.Errorf("failed to parse version number %s for node %s", node.Version, err)
		}
		if majorVersion > maxMajorVersion {
			maxMajorVersion = majorVersion
		}
	}

	return maxMajorVersion, nil
}

// NodeInfoNode represents information about a single node
type nodeInfoNode struct {
	Version string `json:"version"`
}
