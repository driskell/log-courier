package event

import (
	"context"
	"encoding/json"
	"testing"
)

var benchEventJSON = []byte(`{"@timestamp":"2026-10-26T09:15:02.123Z","message":"10.0.0.12 - - [26/Oct/2026:09:15:02 +0000] \"GET /api/v1/items?page=2 HTTP/1.1\" 200 5123 \"-\" \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36\"","host":"web-01","path":"/var/log/nginx/access.log","offset":123456789,"type":"nginx","tags":["prod","web"],"fields":{"env":"prod","team":"platform"}}`)

// BenchmarkEventBytes encodes one already decoded event to JSON per op
func BenchmarkEventBytes(b *testing.B) {
	var data map[string]interface{}
	if err := json.Unmarshal(benchEventJSON, &data); err != nil {
		b.Fatal(err)
	}
	NewEvent(context.Background(), nil, data)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evnt := &Event{data: data}
		evnt.Bytes()
	}
}

// BenchmarkEventDataFromBytes decodes one raw JSON event into its data per op
func BenchmarkEventDataFromBytes(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NewEventFromBytes(context.Background(), nil, benchEventJSON).Data()
	}
}
