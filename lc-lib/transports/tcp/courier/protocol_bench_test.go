package courier

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net"
	"sync"
	"testing"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports/tcp"
	"gopkg.in/op/go-logging.v1"
)

// silenceLoggingOnce avoids replacing the backend while a previous run is still logging
var silenceLoggingOnce sync.Once

var benchEventJSON = []byte(`{"@timestamp":"2026-10-26T09:15:02.123Z","message":"10.0.0.12 - - [26/Oct/2026:09:15:02 +0000] \"GET /api/v1/items?page=2 HTTP/1.1\" 200 5123 \"-\" \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36\"","host":"web-01","path":"/var/log/nginx/access.log","offset":123456789,"type":"nginx","tags":["prod","web"],"fields":{"env":"prod","team":"platform"}}`)

func silenceLogging() {
	silenceLoggingOnce.Do(func() {
		backend := logging.AddModuleLevel(logging.NewLogBackend(io.Discard, "", 0))
		backend.SetLevel(logging.ERROR, "")
		logging.SetBackend(backend)
	})
}

// benchConnection records what is written to it, discards sent messages, and reads from a preset buffer
type benchConnection struct {
	written    bytes.Buffer
	reader     bytes.Reader
	windowSize int
}

func (c *benchConnection) Context() context.Context                      { return context.Background() }
func (c *benchConnection) Write(data []byte) (int, error)                { return c.written.Write(data) }
func (c *benchConnection) Flush() error                                  { return nil }
func (c *benchConnection) Read(data []byte) (int, error)                 { return c.reader.Read(data) }
func (c *benchConnection) ReadByte() (byte, error)                       { return c.reader.ReadByte() }
func (c *benchConnection) SendMessage(message tcp.ProtocolMessage) error { return nil }
func (c *benchConnection) ReceiveWindowSize() int                        { return c.windowSize }
func (c *benchConnection) LocalAddr() net.Addr                           { return &net.TCPAddr{} }
func (c *benchConnection) RemoteAddr() net.Addr                          { return &net.TCPAddr{} }

// BenchmarkProtocolSendEventsEVNT hands one payload of encoded events to the connection as an EVNT message per op
// excluding compression and the socket
func BenchmarkProtocolSendEventsEVNT(b *testing.B) {
	benchmarkProtocolSendEvents(b, true, true)
}

// BenchmarkProtocolSendEventsJDAT hands one payload of encoded events to the connection as a JDAT message per op
// excluding compression and the socket
func BenchmarkProtocolSendEventsJDAT(b *testing.B) {
	benchmarkProtocolSendEvents(b, false, true)
}

// BenchmarkProtocolSendEventsEVNTUnencoded hands one payload of unencoded events to the connection as an EVNT message
// per op, including encoding each event but excluding compression and the socket
func BenchmarkProtocolSendEventsEVNTUnencoded(b *testing.B) {
	benchmarkProtocolSendEvents(b, true, false)
}

// benchmarkProtocolSendEvents sends one 1024 event payload per op, with each event either already encoded as a
// spooler leaves it, or unencoded as a processor leaves it
func benchmarkProtocolSendEvents(b *testing.B, supportsEvnt bool, encoded bool) {
	silenceLogging()

	events := make([]*event.Event, 1024)
	for idx := range events {
		if encoded {
			events[idx] = event.NewEventFromBytes(context.Background(), nil, benchEventJSON)
			continue
		}
		var data map[string]interface{}
		if err := json.Unmarshal(benchEventJSON, &data); err != nil {
			b.Fatal(err)
		}
		events[idx] = event.NewEvent(context.Background(), nil, data)
	}
	p := &protocol{conn: &benchConnection{}, isClient: true, supportsEvnt: supportsEvnt}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// SendEvents caches each encoding on the event, so clear it to encode again every op
		if !encoded {
			for _, evnt := range events {
				evnt.ClearCache()
			}
		}
		if err := p.SendEvents("0123456789abcdef", events); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProtocolReadEVNT reads one 1024 event EVNT message from the connection per op, including decompression
// but excluding the socket; MB/s is the rate of decompressed event data
func BenchmarkProtocolReadEVNT(b *testing.B) {
	silenceLogging()

	events := make([][]byte, 1024)
	for idx := range events {
		events[idx] = benchEventJSON
	}
	nonce := "0123456789abcdef"
	writer := &benchConnection{}
	if err := (&protocolEVNT{nonce: &nonce, events: events}).Write(writer); err != nil {
		b.Fatal(err)
	}
	// The message type and length header is read before the message is constructed
	body := writer.written.Bytes()[8:]
	reader := &benchConnection{windowSize: 1 << 20}

	b.SetBytes(int64(len(benchEventJSON) * len(events)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader.reader.Reset(body)
		message, err := newProtocolEVNT(reader, math.MaxUint32)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := readTestEvents(message.(tcp.StreamMessage)); err != nil {
			b.Fatal(err)
		}
	}
}
