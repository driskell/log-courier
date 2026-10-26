package pipelinebench

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

// sinkConfig describes the simulated output: each routine encodes a payload then waits payloadLatency
type sinkConfig struct {
	routines       int
	payloadLatency time.Duration
}

// pipelineConfig describes the configuration differences between scenarios
type pipelineConfig struct {
	sink                       sinkConfig
	receiverMaxPendingPayloads int
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

	dir := tb.TempDir()
	path := filepath.Join(dir, "log-carver.yaml")
	yaml := fmt.Sprintf(`general:
  persist directory: %q
admin:
  enabled: false
receivers:
- transport: %s
  listen: [bench]
  max pending payloads: %d
network:
  transport: %s
  servers: [127.0.0.1]
  routines: %d
  payload latency: %s
`, dir, benchTransport, cfg.receiverMaxPendingPayloads, benchTransport, cfg.sink.routines, cfg.sink.payloadLatency)
	if err := os.WriteFile(path, []byte(yaml), 0600); err != nil {
		tb.Fatal(err)
	}
	loaded := config.NewConfig()
	if err := loaded.Load(path, true); err != nil {
		tb.Fatal(err)
	}

	app := core.NewAppWithConfig("pipelinebench", "0", loaded)
	app.Pipeline().AddSource(receiver.NewPool(app))
	app.Pipeline().AddProcessor(spooler.NewSpooler(app))
	app.Pipeline().AddProcessor(processor.NewPool(app))
	app.Pipeline().SetSink(publisher.NewPublisher())

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
		acks:     make(chan struct{}, depth*2),
		drops:    make(chan string, depth+1),
	}
	conn.ctx = context.WithValue(context.WithValue(context.Background(), transports.ContextReceiver, transports.Receiver(p.receiver)), transports.ContextConnection, conn)
	p.receiver.eventChan <- transports.NewConnectEvent(conn.ctx, name, benchTransport)
	return conn
}
