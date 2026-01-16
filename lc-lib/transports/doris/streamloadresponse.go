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

package doris

import (
	"encoding/json"
	"fmt"
)

// streamLoadResponse represents the response from a Doris stream load request
type streamLoadResponse struct {
	TxnID                  int64  `json:"TxnId"`
	Label                  string `json:"Label"`
	Comment                string `json:"Comment"`
	TwoPhaseCommit         string `json:"TwoPhaseCommit"`
	Status                 string `json:"Status"`
	Message                string `json:"Message"`
	NumberTotalRows        int    `json:"NumberTotalRows"`
	NumberLoadedRows       int    `json:"NumberLoadedRows"`
	NumberFilteredRows     int    `json:"NumberFilteredRows"`
	NumberUnselectedRows   int    `json:"NumberUnselectedRows"`
	LoadBytes              int64  `json:"LoadBytes"`
	LoadTimeMs             int    `json:"LoadTimeMs"`
	BeginTxnTimeMs         int    `json:"BeginTxnTimeMs"`
	StreamLoadPutTimeMs    int    `json:"StreamLoadPutTimeMs"`
	ReadDataTimeMs         int    `json:"ReadDataTimeMs"`
	WriteDataTimeMs        int    `json:"WriteDataTimeMs"`
	ReceiveDataTimeMs      int    `json:"ReceiveDataTimeMs"`
	CommitAndPublishTimeMs int    `json:"CommitAndPublishTimeMs"`
	ErrorURL               string `json:"ErrorURL"`
	FirstErrorMessage      string `json:"FirstErrorMessage"`
}

// newStreamLoadResponse parses a stream load response
// Note: Doris stream load is atomic - either all events succeed or all fail
func newStreamLoadResponse(body []byte) (*streamLoadResponse, error) {
	response := &streamLoadResponse{}
	if err := json.Unmarshal(body, response); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %s", err)
	}

	return response, nil
}
