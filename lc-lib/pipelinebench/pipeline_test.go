package pipelinebench

import (
	"context"
	"os"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/processor"
	"github.com/driskell/log-courier/lc-lib/publisher"
	"github.com/driskell/log-courier/lc-lib/receiver"
	"github.com/driskell/log-courier/lc-lib/spooler"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// sinkConfig describes the simulated output: each routine encodes a payload then waits payloadLatency,
// or slowLatency for one in every slowEvery payloads when set
type sinkConfig struct {
	routines       int
	payloadLatency time.Duration
	slowEvery      int
	slowLatency    time.Duration
}

// pipelineConfig describes the configuration differences between scenarios
type pipelineConfig struct {
	sink sinkConfig
	// maxQueueSize overrides the receiver queue budget in bytes when set
	maxQueueSize int64
}

// benchPipeline is a running in-process log-carver pipeline
type benchPipeline struct {
	app      *core.App
	receiver *benchReceiver
	done     chan struct{}
}

// startPipeline starts a pipeline wired as log-carver/log-carver.go wires it
func startPipeline(tb testing.TB, cfg pipelineConfig) *benchPipeline {
	tb.Helper()

	general := map[string]interface{}{"persist directory": tb.TempDir()}
	if cfg.maxQueueSize > 0 {
		general["max queue size"] = int(cfg.maxQueueSize)
	}
	loaded := config.NewConfig()
	err := loaded.Parse(map[string]interface{}{
		"general": general,
		"admin":   map[string]interface{}{"enabled": false},
		"receivers": []interface{}{
			map[string]interface{}{"transport": benchTransport, "listen": []interface{}{"bench"}},
		},
		"network": map[string]interface{}{
			"transport":       benchTransport,
			"servers":         []interface{}{"127.0.0.1"},
			"routines":        cfg.sink.routines,
			"payload latency": cfg.sink.payloadLatency.String(),
			"slow every":      cfg.sink.slowEvery,
			"slow latency":    cfg.sink.slowLatency.String(),
		},
	}, true)
	if err != nil {
		tb.Fatal(err)
	}

	app := core.NewAppWithConfig("pipelinebench", "0", loaded)
	app.Pipeline().AddSource(receiver.NewPool(app))
	app.Pipeline().AddProcessor(spooler.NewSpooler(app))
	app.Pipeline().AddProcessor(processor.NewPool(app))
	app.Pipeline().SetSink(publisher.NewPublisher(publisher.UnorderedAcks()))

	p := &benchPipeline{app: app, done: make(chan struct{})}
	signalChan := make(chan os.Signal, 1)
	receiverChan := make(chan *benchReceiver, 1)
	receiverCreated.Store(receiverChan)
	go func() {
		app.Pipeline().Run(loaded, signalChan)
		close(p.done)
	}()

	select {
	case p.receiver = <-receiverChan:
	case <-signalChan:
		tb.Fatal("pipeline failed to start")
	case <-time.After(10 * time.Second):
		tb.Fatal("pipeline did not create the receiver")
	}
	return p
}

// stop shuts down the pipeline, which requires every connection to have disconnected
func (p *benchPipeline) stop(tb testing.TB) {
	tb.Helper()
	p.app.Pipeline().Shutdown()
	select {
	case <-p.done:
	case <-time.After(30 * time.Second):
		tb.Fatalf("pipeline did not shut down:\n%s", goroutineStacks())
	}
}

// goroutineStacks returns the stacks of all goroutines, to show where a hang occurred
func goroutineStacks() string {
	var stacks strings.Builder
	pprof.Lookup("goroutine").WriteTo(&stacks, 1)
	return stacks.String()
}

// connect opens a simulated client connection that sends payloads of payloadEvents events
func (p *benchPipeline) connect(name string, payloadEvents int, depth int) *benchConnection {
	conn := &benchConnection{
		name:     name,
		receiver: p.receiver,
		payload:  newBenchPayload(payloadEvents),
		depth:    depth,
		acks:     make(chan string, depth*2),
		closed:   make(chan struct{}, 1),
	}
	conn.ctx = context.WithValue(context.WithValue(context.Background(), transports.ContextReceiver, transports.Receiver(p.receiver)), transports.ContextConnection, conn)
	general := p.app.Config().General()
	conn.window = transports.NewReceiveWindow(conn.ctx, int(general.ReceiveWindowSize), general.ReceiveWindowFill)
	p.receiver.eventChan <- transports.NewConnectEvent(conn.ctx, name, benchTransport)
	return conn
}
