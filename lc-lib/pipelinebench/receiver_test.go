package pipelinebench

import (
	"context"
	"sync/atomic"

	"github.com/driskell/log-courier/lc-lib/transports"
)

// receiverCreated hands the receiver created by the pool to the pipeline that started it
var receiverCreated atomic.Value

type benchReceiverFactory struct{}

func (f *benchReceiverFactory) NewReceiver(ctx context.Context, listen string, eventChan chan<- transports.Event) transports.Receiver {
	r := &benchReceiver{eventChan: eventChan}
	if receiverChan := receiverCreated.Load(); receiverChan != nil {
		receiverChan.(chan *benchReceiver) <- r
	}
	return r
}

func (f *benchReceiverFactory) ShouldRestart(transports.ReceiverFactory) bool { return false }

// benchReceiver lets simulated connections feed the receiver pool directly
type benchReceiver struct {
	eventChan chan<- transports.Event
}

func (r *benchReceiver) Acknowledge(ctx context.Context, nonce *string, sequence uint32) error {
	ctx.Value(transports.ContextConnection).(*benchConnection).acknowledge(*nonce, sequence)
	return nil
}

func (r *benchReceiver) Pong(context.Context) error { return nil }

func (r *benchReceiver) FailConnection(ctx context.Context, err error) {
	ctx.Value(transports.ContextConnection).(*benchConnection).close()
}

func (r *benchReceiver) ShutdownConnection(context.Context) {}

func (r *benchReceiver) ShutdownConnectionRead(ctx context.Context, err error) {
	ctx.Value(transports.ContextConnection).(*benchConnection).close()
}

// Shutdown reports the receiver finished from another routine as the pool calls this from its own loop
func (r *benchReceiver) Shutdown() {
	ctx := context.WithValue(context.Background(), transports.ContextReceiver, transports.Receiver(r))
	go func() {
		r.eventChan <- transports.NewStatusEvent(ctx, transports.Finished, nil)
	}()
}

func (r *benchReceiver) Factory() transports.ReceiverFactory { return &benchReceiverFactory{} }

func (r *benchReceiver) SupportsAck() bool { return true }
