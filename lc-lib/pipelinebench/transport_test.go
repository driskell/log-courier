package pipelinebench

import (
	"context"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// benchTransportFactory configures the simulated output transport
type benchTransportFactory struct {
	Routines       int           `config:"routines"`
	PayloadLatency time.Duration `config:"payload latency"`
}

func newBenchTransportFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (transports.TransportFactory, error) {
	ret := &benchTransportFactory{}
	if err := p.Populate(ret, unUsed, configPath, true); err != nil {
		return nil, err
	}
	return ret, nil
}

func (f *benchTransportFactory) NewTransport(ctx context.Context, poolEntry *addresspool.PoolEntry, eventChan chan<- transports.Event) transports.Transport {
	t := &benchTransportImpl{
		ctx:       ctx,
		factory:   f,
		eventChan: eventChan,
		payloads:  make(chan benchTransportPayload, 1024),
	}
	for i := 0; i < max(f.Routines, 1); i++ {
		t.group.Add(1)
		go t.routine()
	}
	eventChan <- transports.NewStatusEvent(ctx, transports.Started, nil)
	return t
}

func (f *benchTransportFactory) ShouldRestart(transports.TransportFactory) bool { return false }

type benchTransportPayload struct {
	nonce  string
	events []*event.Event
}

// benchTransportImpl encodes each payload as a real output would, waits the configured latency, then acknowledges it
type benchTransportImpl struct {
	ctx       context.Context
	factory   *benchTransportFactory
	eventChan chan<- transports.Event
	payloads  chan benchTransportPayload
	group     sync.WaitGroup
	closeOnce sync.Once
}

func (t *benchTransportImpl) routine() {
	defer t.group.Done()
	for payload := range t.payloads {
		for _, evnt := range payload.events {
			evnt.Bytes()
		}
		if t.factory.PayloadLatency > 0 {
			time.Sleep(t.factory.PayloadLatency)
		}
		nonce := payload.nonce
		t.eventChan <- transports.NewAckEvent(t.ctx, &nonce, uint32(len(payload.events)))
	}
}

func (t *benchTransportImpl) SendEvents(nonce string, events []*event.Event) error {
	t.payloads <- benchTransportPayload{nonce: nonce, events: events}
	return nil
}

func (t *benchTransportImpl) Ping() error {
	go func() {
		t.eventChan <- transports.NewPongEvent(t.ctx)
	}()
	return nil
}

func (t *benchTransportImpl) Fail() {
	t.Shutdown()
}

func (t *benchTransportImpl) Shutdown() {
	t.closeOnce.Do(func() {
		close(t.payloads)
		go func() {
			t.group.Wait()
			t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Finished, nil)
		}()
	})
}

func (t *benchTransportImpl) Factory() transports.TransportFactory { return t.factory }
