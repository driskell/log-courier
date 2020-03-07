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

package es

import (
	"strings"
	"testing"
)

func TestBulkResponseParse(t *testing.T) {
	request := createTestBulkRequest(3, "2020-08-03", "2020-08-03")
	responseString := `{"ignore":1,"again":"test","took":444,"more":false,"items":[{"index":{"in":5,"result":"created"}},{"index":{"result":"failed","out":false}},{"index":{"result":"created"}}],"last":"gone"}`
	response, err := newBulkResponse(strings.NewReader(responseString), request)
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

func TestBulkResponseParseTooFew(t *testing.T) {
	request := createTestBulkRequest(3, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"items":[{"index":{"result":"created"}}]}`
	_, err := newBulkResponse(strings.NewReader(responseString), request)
	if err == nil {
		t.Errorf("Unexpected response parse success")
	}
}

func TestBulkResponseParseTookMissing(t *testing.T) {
	request := createTestBulkRequest(3, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"items":[{"index":{"result":"created"}},{"index":{"result":"failed"}},{"index":{"result":"created"}}]}`
	response, err := newBulkResponse(strings.NewReader(responseString), request)
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
	_, err := newBulkResponse(strings.NewReader(responseString), request)
	if err == nil {
		t.Errorf("Unexpected response parse success")
	}
}

func TestBulkResponseParseSyntax(t *testing.T) {
	request := createTestBulkRequest(1, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"items":[{"index":{"result":"created"}}]}`
	_, err := newBulkResponse(strings.NewReader(responseString), request)
	if err != nil {
		t.Errorf("Unexpected response parse error: %s", err)
	}
	responseString = `{"more":false,"items":["index":{"result":"created"}}]}`
	_, err = newBulkResponse(strings.NewReader(responseString), request)
	if err == nil {
		t.Errorf("Unexpected response parse success")
	}
}
