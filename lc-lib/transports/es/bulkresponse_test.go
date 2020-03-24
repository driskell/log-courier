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
	"testing"
)

func TestBulkResponseParse(t *testing.T) {
	request := createTestBulkRequest(3, "2020-08-03", "2020-08-03")
	responseString := `{"ignore":1,"again":"test","took":444,"more":false,"items":[{"index":{"in":5,"result":"created"}},{"index":{"error":{"type": "failed"},"status":404}},{"index":{"result":"created"}}],"last":"gone"}`
	response, err := newBulkResponse([]byte(responseString), request)
	if err != nil {
		t.Errorf("Unexpected response parse error: %s", err)
	}
	if request.Remaining() != 1 {
		t.Errorf("Unexpected remaining count: %d", request.Remaining())
	}
	if response.Took != 444 {
		t.Errorf("Unexpected took value: %d", response.Took)
	}
}

func TestBulkResponseParseSkip(t *testing.T) {
	request := createTestBulkRequest(3, "2020-08-03", "2020-08-03")
	responseString := `{"ignore":1,"again":"test","took":222,"more":false,"items":[{"index":{"in":5,"result":"created"}},{"index":{"error":{"type": "failed"},"status":400}},{"index":{"result":"created"}}],"last":"gone"}`
	response, err := newBulkResponse([]byte(responseString), request)
	if err != nil {
		t.Errorf("Unexpected response parse error: %s", err)
	}
	if request.Remaining() != 0 {
		t.Errorf("Unexpected remaining count: %d", request.Remaining())
	}
	if response.Took != 222 {
		t.Errorf("Unexpected took value: %d", response.Took)
	}
}

func TestBulkResponseParseTooFew(t *testing.T) {
	request := createTestBulkRequest(3, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"items":[{"index":{"result":"created"}}]}`
	_, err := newBulkResponse([]byte(responseString), request)
	if err == nil {
		t.Errorf("Unexpected response parse success")
	}
}

func TestBulkResponseParseTookMissing(t *testing.T) {
	request := createTestBulkRequest(3, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"items":[{"index":{"result":"created"}},{"index":{"status":500,"error":{"type":"error"}}},{"index":{"result":"created"}}]}`
	response, err := newBulkResponse([]byte(responseString), request)
	if err != nil {
		t.Errorf("Unexpected response parse error: %s", err)
	}
	if request.Remaining() != 1 {
		t.Errorf("Unexpected remaining count: %d", request.Remaining())
	}
	if response.Took != 0 {
		t.Errorf("Unexpected took value: %d", response.Took)
	}
}

func TestBulkResponseParseTooMany(t *testing.T) {
	request := createTestBulkRequest(1, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"items":[{"index":{"result":"created"}},{"index":{"result":"failed"}},{"index":{"result":"created"}}]}`
	_, err := newBulkResponse([]byte(responseString), request)
	if err == nil {
		t.Errorf("Unexpected response parse success")
	}
}

func TestBulkResponseParseSyntax(t *testing.T) {
	request := createTestBulkRequest(1, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"items":[{"index":{"result":"created"}}]}`
	_, err := newBulkResponse([]byte(responseString), request)
	if err != nil {
		t.Errorf("Unexpected response parse error: %s", err)
	}
	responseString = `{"more":false,"items":["index":{"result":"created"}}]}`
	_, err = newBulkResponse([]byte(responseString), request)
	if err == nil {
		t.Errorf("Unexpected response parse success")
	}
}
