package pipelinebench

import (
	"testing"
	"time"
)

func TestPipelineStartsWithEachScenarioConfiguration(t *testing.T) {
	for _, cfg := range []pipelineConfig{
		{sink: sinkConfig{routines: 4}},
		{sink: sinkConfig{routines: 4, payloadLatency: 10 * time.Millisecond, slowEvery: 50, slowLatency: 250 * time.Millisecond}, maxQueueSize: 4 << 20},
	} {
		startPipeline(t, cfg).stop(t)
	}
}
