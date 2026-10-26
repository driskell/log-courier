package spooler

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"gopkg.in/op/go-logging.v1"
)

const (
	benchBatchSize = 128
	benchRingSize  = 64
)

// silenceLoggingOnce avoids replacing the backend while a previous run is still logging
var silenceLoggingOnce sync.Once

var benchEventJSON = []byte(`{"@timestamp":"2026-10-26T09:15:02.123Z","message":"10.0.0.12 - - [26/Oct/2026:09:15:02 +0000] \"GET /api/v1/items?page=2 HTTP/1.1\" 200 5123 \"-\" \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36\"","host":"web-01","path":"/var/log/nginx/access.log","offset":123456789,"type":"nginx","tags":["prod","web"],"fields":{"env":"prod","team":"platform"}}`)

// BenchmarkSpoolerEncodedEvents spools one batch of benchBatchSize events that already hold their encoding per op,
// showing the spooler's own overhead; events/s is the per event rate
func BenchmarkSpoolerEncodedEvents(b *testing.B) {
	benchmarkSpooler(b, func() *event.Event {
		return event.NewEventFromBytes(context.Background(), nil, benchEventJSON)
	}, false)
}

// BenchmarkSpoolerUnencodedEvents spools one batch of benchBatchSize events that hold no encoding per op,
// including any encoding the spooler performs to size them; events/s is the per event rate
func BenchmarkSpoolerUnencodedEvents(b *testing.B) {
	benchmarkSpooler(b, func() *event.Event {
		var data map[string]interface{}
		if err := json.Unmarshal(benchEventJSON, &data); err != nil {
			b.Fatal(err)
		}
		return event.NewEvent(context.Background(), nil, data)
	}, true)
}

// benchmarkSpooler feeds a ring of reusable batches through the spooler, where
// clearCache resets each spooled event so it is unencoded again when reused
func benchmarkSpooler(b *testing.B, newEvent func() *event.Event, clearCache bool) {
	silenceLoggingOnce.Do(func() {
		backend := logging.AddModuleLevel(logging.NewLogBackend(io.Discard, "", 0))
		backend.SetLevel(logging.ERROR, "")
		logging.SetBackend(backend)
	})

	ring := make([][]*event.Event, benchRingSize)
	for idx := range ring {
		ring[idx] = make([]*event.Event, benchBatchSize)
		for eventIdx := range ring[idx] {
			ring[idx][eventIdx] = newEvent()
		}
	}

	spooler := NewSpooler(core.NewAppWithConfig("bench", "0", config.NewConfig()))
	output := make(chan []*event.Event, 1)
	shutdownChan := make(chan struct{})
	spooler.SetOutput(output)
	spooler.SetShutdownChan(shutdownChan)
	spooler.SetConfigChan(make(chan *config.Config))
	go spooler.Run()

	// Batches return to the free slots in order once every event in them was consumed
	freeSlots := make(chan struct{}, benchRingSize)
	for idx := 0; idx < benchRingSize; idx++ {
		freeSlots <- struct{}{}
	}
	consumed := make(chan int)
	go func() {
		total := 0
		for spool := range output {
			if clearCache {
				for _, evnt := range spool {
					evnt.ClearCache()
				}
			}
			total += len(spool)
			for ; total >= benchBatchSize; total -= benchBatchSize {
				freeSlots <- struct{}{}
			}
		}
		close(consumed)
	}()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		<-freeSlots
		spooler.Input() <- ring[i%benchRingSize]
	}
	b.StopTimer()
	b.ReportMetric(float64(b.N*benchBatchSize)/b.Elapsed().Seconds(), "events/s")

	close(shutdownChan)
	<-consumed
}
