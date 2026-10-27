package pipelinebench

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkPipelineThroughput sends one event per op through the whole pipeline to an output that acknowledges
// instantly, with b.N events split across the connections; similar results across connection counts indicate a
// serial stage limits throughput
func BenchmarkPipelineThroughput(b *testing.B) {
	for _, connections := range []int{1, 100} {
		b.Run(fmt.Sprintf("connections=%d", connections), func(b *testing.B) {
			const payloadEvents, depth = 1024, 10

			p := startPipeline(b, pipelineConfig{
				sink: sinkConfig{routines: 4},
			})
			conns := make([]*benchConnection, connections)
			for idx := range conns {
				conns[idx] = p.connect(fmt.Sprintf("conn%d", idx), payloadEvents, depth)
			}

			payloads := (b.N + payloadEvents - 1) / payloadEvents
			b.ReportAllocs()
			b.ResetTimer()
			results := runConnections(b, conns, func(idx int) int {
				quota := payloads / connections
				if idx < payloads%connections {
					quota++
				}
				return quota
			}, time.Now().Add(time.Hour))
			b.StopTimer()

			events, closes := 0, 0
			for _, result := range results {
				if result.closed {
					closes++
				}
				events += result.events
			}
			// Closed connections deliver fewer than b.N events, so time is reported per delivered event
			if events != 0 {
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(events), "ns/op")
			}
			b.ReportMetric(float64(events)/b.Elapsed().Seconds(), "events/s")
			b.ReportMetric(float64(closes), "closes")
			p.stop(b)
		})
	}
}
