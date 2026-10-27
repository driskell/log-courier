package pipelinebench

import (
	"context"
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

// benchConnection simulates a client that keeps up to depth payloads in flight
type benchConnection struct {
	ctx      context.Context
	name     string
	receiver *benchReceiver
	payload  [][]byte
	depth    int
	window   *transports.ReceiveWindow
	acks     chan string
	// closed is signalled when the receiver closes the connection
	closed chan struct{}
	nonce  int
}

type inflightPayload struct {
	nonce    string
	sent     time.Time
	recorded bool
}

// run sends up to maxPayloads payloads, recording those sent before the deadline, and marks recorded done once
// they are all acknowledged; it keeps the load steady with unrecorded payloads until stop closes, then drains and disconnects
// A connection closed by the receiver stops sending and abandons its payloads in flight
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
	// Acknowledgements are matched by nonce, as payloads may be acknowledged in any order
	remove := func(nonce string) (inflightPayload, bool) {
		for idx, entry := range inflight {
			if entry.nonce == nonce {
				if entry.recorded {
					recordedInflight--
				}
				inflight = append(inflight[:idx], inflight[idx+1:]...)
				return entry, true
			}
		}
		return inflightPayload{}, false
	}
	onClosed := func() {
		result.closed = true
		inflight = inflight[:0]
		recordedInflight = 0
	}

	for {
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
			sent++
			if recording {
				recordedInflight++
			}
			if !c.send(nonce) {
				onClosed()
			}
			continue
		}

		select {
		case nonce := <-c.acks:
			acked := time.Now()
			if entry, ok := remove(nonce); ok && entry.recorded {
				result.latencies = append(result.latencies, acked.Sub(entry.sent))
				if acked.Before(deadline) {
					result.events += len(c.payload)
				}
			}
		case <-stopChan:
			stopChan = nil
		case <-c.closed:
			onClosed()
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

// acknowledge is called by the receiver pool and must never block it; only a payload's full acknowledgement completes it
func (c *benchConnection) acknowledge(nonce string, sequence uint32) {
	if int(sequence) != len(c.payload) {
		return
	}
	select {
	case c.acks <- nonce:
	default:
	}
}

// close reports the receiver closed the connection; it must never block the receiver pool
func (c *benchConnection) close() {
	select {
	case c.closed <- struct{}{}:
	default:
	}
}

// send holds each event of a payload in the receive window as a receiving connection does, waiting while the window
// is full, returning false if the receiver closes the connection meanwhile
func (c *benchConnection) send(nonce string) bool {
	for _, data := range c.payload {
		if !c.window.Fits(len(data)) && c.window.Request() && !c.requestTake() {
			return false
		}
		for !c.window.Fits(len(data)) {
			select {
			case <-c.window.Taken():
			case <-c.closed:
				return false
			}
		}
		if c.window.AddRaw(&nonce, data) && !c.requestTake() {
			return false
		}
	}
	return !c.window.End(&nonce) || c.requestTake()
}

// requestTake asks the receiver pool to take the events held in the receive window
func (c *benchConnection) requestTake() bool {
	select {
	case c.receiver.eventChan <- transports.NewAdmissionEvent(c.ctx, c.window):
		return true
	case <-c.closed:
		return false
	}
}
