package pipelinebench

import (
	"fmt"
	"math"
	"testing"
	"time"
)

// fairnessScenario mixes a few heavy senders with many light ones against an output with real latency
type fairnessScenario struct {
	heavyConnections, heavyPayloadEvents, heavyDepth int
	lightConnections, lightPayloadEvents             int
}

// BenchmarkPipelineFairness runs one fixed duration scenario per op, of a few heavy senders and many light ones against
// an output with latency, reporting throughput and acknowledgement latency per sender class; use -benchtime 1x
func BenchmarkPipelineFairness(b *testing.B) {
	duration := 5 * time.Second
	if testing.Short() {
		duration = time.Second
	}
	for _, scenario := range []fairnessScenario{
		{heavyConnections: 5, heavyPayloadEvents: 1024, heavyDepth: 1, lightConnections: 295, lightPayloadEvents: 10},
		{heavyConnections: 5, heavyPayloadEvents: 1024, heavyDepth: 10, lightConnections: 295, lightPayloadEvents: 10},
	} {
		name := fmt.Sprintf("heavy=%dx%dx%d/light=%dx%d", scenario.heavyConnections, scenario.heavyPayloadEvents, scenario.heavyDepth, scenario.lightConnections, scenario.lightPayloadEvents)
		b.Run(name, func(b *testing.B) {
			var heavyEvents, lightEvents, closes int
			var heavyLatencies, lightLatencies []time.Duration
			var lightRates []float64
			for i := 0; i < b.N; i++ {
				p := startPipeline(b, pipelineConfig{
					sink:                       sinkConfig{routines: 4, payloadLatency: 10 * time.Millisecond},
					receiverMaxPendingPayloads: 10,
				})
				heavy := make([]*benchConnection, scenario.heavyConnections)
				for idx := range heavy {
					heavy[idx] = p.connect(fmt.Sprintf("heavy%d", idx), scenario.heavyPayloadEvents, scenario.heavyDepth)
				}
				light := make([]*benchConnection, scenario.lightConnections)
				for idx := range light {
					light[idx] = p.connect(fmt.Sprintf("light%d", idx), scenario.lightPayloadEvents, 1)
				}

				results := runConnections(b, append(heavy, light...), func(int) int { return math.MaxInt }, time.Now().Add(duration))
				p.stop(b)

				for idx, result := range results {
					if result.closed {
						closes++
					}
					if idx < len(heavy) {
						heavyEvents += result.events
						heavyLatencies = append(heavyLatencies, result.latencies...)
						continue
					}
					lightEvents += result.events
					lightLatencies = append(lightLatencies, result.latencies...)
					lightRates = append(lightRates, float64(result.events)/duration.Seconds())
				}
			}

			sortDurations(heavyLatencies)
			sortDurations(lightLatencies)
			seconds := duration.Seconds() * float64(b.N)
			b.ReportMetric(0, "ns/op")
			b.ReportMetric(float64(heavyEvents+lightEvents)/seconds, "events/s")
			b.ReportMetric(float64(heavyEvents)/seconds, "heavy-events/s")
			b.ReportMetric(float64(lightEvents)/seconds, "light-events/s")
			b.ReportMetric(float64(percentile(heavyLatencies, 0.5))/float64(time.Millisecond), "heavy-p50-ms")
			b.ReportMetric(float64(percentile(lightLatencies, 0.5))/float64(time.Millisecond), "light-p50-ms")
			b.ReportMetric(float64(percentile(lightLatencies, 0.99))/float64(time.Millisecond), "light-p99-ms")
			b.ReportMetric(float64(percentile(lightLatencies, 1))/float64(time.Millisecond), "light-max-ms")
			b.ReportMetric(jainIndex(lightRates), "light-jain")
			b.ReportMetric(float64(closes)/float64(b.N), "closes")
		})
	}
}
