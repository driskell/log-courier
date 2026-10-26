package pipelinebench

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/driskell/log-courier/lc-lib/transports"
)

// newBenchPayload builds raw JSON events with current timestamps so the timestamp guard keeps them
func newBenchPayload(count int) [][]byte {
	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	payload := make([][]byte, count)
	for idx := range payload {
		payload[idx] = []byte(fmt.Sprintf(`{"@timestamp":%q,"message":"10.0.0.%d - - [26/Oct/2026:09:15:02 +0000] \"GET /api/v1/items?page=%d HTTP/1.1\" 200 5123 \"-\" \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36\"","host":"web-01","path":"/var/log/nginx/access.log","offset":%d,"type":"nginx","tags":["prod","web"],"fields":{"env":"prod","team":"platform"}}`, timestamp, idx%256, idx, idx*256))
	}
	return payload
}

// connectionResult records what a simulated connection achieved
type connectionResult struct {
	events    int
	latencies []time.Duration
	closed    bool
}

// benchContextKey is a context key for values the harness attaches to payloads
type benchContextKey string

// contextPayloadNonce identifies the payload a receiver call relates to
const contextPayloadNonce benchContextKey = "nonce"

// benchConnection simulates a client that keeps up to depth payloads in flight
type benchConnection struct {
	ctx      context.Context
	name     string
	receiver *benchReceiver
	payload  [][]byte
	depth    int
	acks     chan struct{}
	drops    chan string
	nonce    int
}

type inflightPayload struct {
	nonce    string
	sent     time.Time
	recorded bool
}

// run sends up to maxPayloads payloads, recording those sent before the deadline, and marks recorded done once
// they are all acknowledged; it keeps the load steady with unrecorded payloads until stop closes, then drains and disconnects
// A connection closed by the receiver stops sending and drains the payloads the receiver accepted before disconnecting
func (c *benchConnection) run(maxPayloads int, deadline time.Time, recorded *sync.WaitGroup, stop <-chan struct{}) *connectionResult {
	result := &connectionResult{}
	inflight := make([]inflightPayload, 0, c.depth)
	sent, recordedInflight := 0, 0
	recordedDone := false
	markRecordedDone := func() {
		if !recordedDone {
			recordedDone = true
			recorded.Done()
		}
	}
	defer markRecordedDone()
	stopChan := stop
	onDrop := func(nonce string) {
		result.closed = true
		for idx, entry := range inflight {
			if entry.nonce == nonce {
				if entry.recorded {
					recordedInflight--
				}
				inflight = append(inflight[:idx], inflight[idx+1:]...)
				break
			}
		}
	}
	// Drops are reported before any later payload is accepted, so draining them first keeps acks matched to payloads
	drainDrops := func() {
		for {
			select {
			case nonce := <-c.drops:
				onDrop(nonce)
			default:
				return
			}
		}
	}

	for {
		drainDrops()
		recording := !result.closed && sent < maxPayloads && time.Now().Before(deadline)
		if !recording && recordedInflight == 0 {
			markRecordedDone()
		}

		// Disconnecting with payloads in flight can hang pipeline shutdown, so drain before disconnecting
		stopping := stopChan == nil
		if !stopping {
			select {
			case <-stopChan:
				stopping, stopChan = true, nil
			default:
			}
		}
		if (stopping || result.closed) && len(inflight) == 0 {
			break
		}

		if len(inflight) < c.depth && sent < maxPayloads && !stopping && !result.closed {
			nonce := fmt.Sprintf("%s-%d", c.name, c.nonce)
			c.nonce++
			inflight = append(inflight, inflightPayload{nonce: nonce, sent: time.Now(), recorded: recording})
			ctx := context.WithValue(c.ctx, contextPayloadNonce, nonce)
			select {
			case c.receiver.eventChan <- &benchEventsEvent{ctx: ctx, nonce: &nonce, events: c.payload, size: payloadSize(c.payload)}:
				sent++
				if recording {
					recordedInflight++
				}
			case nonce := <-c.drops:
				inflight = inflight[:len(inflight)-1]
				onDrop(nonce)
			}
			continue
		}

		select {
		case <-c.acks:
			drainDrops()
			acked := time.Now()
			if inflight[0].recorded {
				result.latencies = append(result.latencies, acked.Sub(inflight[0].sent))
				if acked.Before(deadline) {
					result.events += len(c.payload)
				}
				recordedInflight--
			}
			inflight = inflight[1:]
		case <-stopChan:
			stopChan = nil
		case nonce := <-c.drops:
			onDrop(nonce)
		}
	}

	c.receiver.eventChan <- transports.NewDisconnectEvent(c.ctx, c.name, benchTransport)
	return result
}

// runConnectionsTimeout bounds how long connections may run before the harness reports a hang
const runConnectionsTimeout = 5 * time.Minute

// runConnections runs every connection until all recorded payloads are acknowledged, returning when all have disconnected
func runConnections(tb testing.TB, conns []*benchConnection, maxPayloads func(idx int) int, deadline time.Time) []*connectionResult {
	tb.Helper()
	results := make([]*connectionResult, len(conns))
	stop := make(chan struct{})
	done := make(chan struct{})
	var recorded, finished sync.WaitGroup
	for idx, conn := range conns {
		recorded.Add(1)
		finished.Add(1)
		go func(idx int, conn *benchConnection) {
			defer finished.Done()
			results[idx] = conn.run(maxPayloads(idx), deadline, &recorded, stop)
		}(idx, conn)
	}
	go func() {
		recorded.Wait()
		close(stop)
		finished.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(runConnectionsTimeout):
		tb.Fatalf("connections did not finish:\n%s", goroutineStacks())
	}
	return results
}

// acknowledge is called by the receiver pool and must never block it
func (c *benchConnection) acknowledge(sequence uint32) {
	if int(sequence) != len(c.payload) {
		return
	}
	select {
	case c.acks <- struct{}{}:
	default:
	}
}

// drop reports the receiver closed the connection, rejecting the payload with the given nonce if any; it must never block the receiver pool
func (c *benchConnection) drop(nonce string) {
	select {
	case c.drops <- nonce:
	default:
		panic(fmt.Sprintf("connection %s dropped more payloads than it had in flight", c.name))
	}
}

func payloadSize(payload [][]byte) int {
	size := 0
	for _, data := range payload {
		size += len(data)
	}
	return size
}

// benchEventsEvent provides raw events, and decodes them on every Events call, as the courier protocol messages do
type benchEventsEvent struct {
	ctx    context.Context
	nonce  *string
	events [][]byte
	size   int
}

func (e *benchEventsEvent) Context() context.Context { return e.ctx }
func (e *benchEventsEvent) Nonce() *string           { return e.nonce }
func (e *benchEventsEvent) Count() uint32            { return uint32(len(e.events)) }
func (e *benchEventsEvent) Size() int                { return e.size }
func (e *benchEventsEvent) RawEvents() [][]byte      { return e.events }

func (e *benchEventsEvent) Events() []map[string]interface{} {
	events := make([]map[string]interface{}, 0, len(e.events))
	for _, data := range e.events {
		var decoded map[string]interface{}
		if err := json.Unmarshal(data, &decoded); err != nil {
			decoded = map[string]interface{}{"message": err.Error()}
		}
		events = append(events, decoded)
	}
	return events
}
