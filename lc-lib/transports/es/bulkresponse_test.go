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
		t.Fatalf("Unexpected response parse error: %s", err)
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
		t.Fatalf("Unexpected response parse error: %s", err)
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
	responseString := `{"more":false,"took":444,"items":[{"index":{"result":"created"}}]}`
	_, err := newBulkResponse([]byte(responseString), request)
	if err == nil {
		t.Error("Unexpected response parse success")
	}
}

func TestBulkResponseParseTookMissing(t *testing.T) {
	request := createTestBulkRequest(3, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"items":[{"index":{"result":"created"}},{"index":{"status":500,"error":{"type":"error"}}},{"index":{"result":"created"}}]}`
	_, err := newBulkResponse([]byte(responseString), request)
	if err == nil {
		t.Error("Expected missing took key error")
	}
}

func TestBulkResponseParseItemsMissing(t *testing.T) {
	request := createTestBulkRequest(3, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"took":123}`
	_, err := newBulkResponse([]byte(responseString), request)
	if err == nil {
		t.Error("Expected missing items key error")
	}
}

func TestBulkResponseParseTooMany(t *testing.T) {
	request := createTestBulkRequest(1, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"items":[{"index":{"result":"created"}},{"index":{"result":"failed"}},{"index":{"result":"created"}}]}`
	_, err := newBulkResponse([]byte(responseString), request)
	if err == nil {
		t.Error("Expected too many error")
	}
}

func TestBulkResponseParseSyntax(t *testing.T) {
	request := createTestBulkRequest(1, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"took":123,"items":[{"index":{"result":"created"}}]}`
	_, err := newBulkResponse([]byte(responseString), request)
	if err != nil {
		t.Errorf("Unexpected response parse error: %s", err)
	}
	responseString = `{"more":false,"took":123,"items":["index":{"result":"created"}}]}`
	_, err = newBulkResponse([]byte(responseString), request)
	if err == nil {
		t.Error("Expected parse error")
	}
}

func TestBulkResponseDuplicateTook(t *testing.T) {
	request := createTestBulkRequest(1, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"took":123,"took":444,"items":[{"index":{"result":"created"}}]}`
	_, err := newBulkResponse([]byte(responseString), request)
	if err == nil {
		t.Error("Expected duplicate took error")
	}
}

func TestBulkResponseDuplicateItems(t *testing.T) {
	request := createTestBulkRequest(1, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"took":123,"items":[{"index":{"result":"created"}}],"items":[{"index":{"result":"created"}}]}`
	_, err := newBulkResponse([]byte(responseString), request)
	if err == nil {
		t.Error("Expected duplicate items error")
	}
}

func TestBulkResponseExtra(t *testing.T) {
	request := createTestBulkRequest(1, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"took":123,"items":[{"index":{"result":"created"}}]}ExtraStuff`
	_, err := newBulkResponse([]byte(responseString), request)
	if err == nil {
		t.Error("Expected extra data error")
	}
}

func TestBulkResponseIgnoreUnknown(t *testing.T) {
	request := createTestBulkRequest(1, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"took":123,"items":[{"index":{"result":"created"}}],"ignored":123,"ignored":{"test":""},"moreignore":["ignore",{"more":1.2}]}`
	_, err := newBulkResponse([]byte(responseString), request)
	if err != nil {
		t.Errorf("Unexpected parse error with data that should be ignored: %s", err)
	}
}

func TestBulkResponseError(t *testing.T) {
	request := createTestBulkRequest(1, "2020-08-03", "2020-08-03")
	responseString := `{"more":false,"took":123,"items":[{"index":{"error":{"type":"Type","reason":"Error","caused_by":{"type":"Inner","reason":"message"}}}}]}`
	response, err := newBulkResponse([]byte(responseString), request)
	if err != nil {
		t.Fatalf("Unexpected parse error with error response: %s", err)
	}
	if len(response.Errors) != 1 {
		t.Fatalf("Unexpected number of errors: %d", len(response.Errors))
	}
	if response.Errors[0].Type != "Type" || response.Errors[0].Reason != "Error" || response.Errors[0].CausedBy == nil {
		t.Fatal("Unexpected error format")
	}
	if response.Errors[0].CausedBy.Type != "Inner" || response.Errors[0].CausedBy.Reason != "message" || response.Errors[0].CausedBy.CausedBy != nil {
		t.Fatal("Unexpected inner error format")
	}
}
