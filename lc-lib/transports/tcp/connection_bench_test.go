package tcp

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports"
)

var benchEventJSON = []byte(`{"@timestamp":"2026-10-26T09:15:02.123Z","message":"10.0.0.12 - - [26/Oct/2026:09:15:02 +0000] \"GET /api/v1/items?page=2 HTTP/1.1\" 200 5123 \"-\" \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36\"","host":"web-01","path":"/var/log/nginx/access.log","offset":123456789,"type":"nginx","tags":["prod","web"],"fields":{"env":"prod","team":"platform"}}`)

// benchSocket is a real TCP socket, so setting read deadlines costs what it does in production
type benchSocket struct {
	*net.TCPConn
}

func (s *benchSocket) Setup(context.Context) error { return nil }
func (s *benchSocket) Desc() string                { return "bench" }

// benchProtocol is a blocking protocol whose messages are never read or written
type benchProtocol struct{}

func (p *benchProtocol) Negotiation() (transports.Event, error)  { return nil, nil }
func (p *benchProtocol) SendEvents(string, []*event.Event) error { return nil }
func (p *benchProtocol) Ping() error                             { return nil }
func (p *benchProtocol) Pong() error                             { return nil }
func (p *benchProtocol) Acknowledge(*string, uint32) error       { return nil }
func (p *benchProtocol) Read() (transports.Event, error)         { return nil, nil }
func (p *benchProtocol) NonBlocking() bool                       { return false }

// newBenchSocket returns one end of a loopback TCP connection
func newBenchSocket(b *testing.B) *benchSocket {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { listener.Close() })
	accepted := make(chan *net.TCPConn, 1)
	go func() {
		conn, _ := listener.AcceptTCP()
		accepted <- conn
	}()
	conn, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { conn.Close() })
	if remote := <-accepted; remote != nil {
		b.Cleanup(func() { remote.Close() })
	}
	return &benchSocket{TCPConn: conn}
}

// BenchmarkConnectionReadCompressed decompresses one 1024 event payload read through the connection per op,
// as the courier protocol does, with the compressed data already buffered; MB/s is the rate of decompressed data
func BenchmarkConnectionReadCompressed(b *testing.B) {
	var compressed bytes.Buffer
	compressor, err := zlib.NewWriterLevel(&compressed, 3)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 1024; i++ {
		if err := binary.Write(compressor, binary.BigEndian, uint32(len(benchEventJSON))); err != nil {
			b.Fatal(err)
		}
		if _, err := compressor.Write(benchEventJSON); err != nil {
			b.Fatal(err)
		}
	}
	if err := compressor.Close(); err != nil {
		b.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.Cleanup(cancel)
	source := bytes.NewReader(compressed.Bytes())
	conn := &connection{
		ctx:                  ctx,
		socket:               newBenchSocket(b),
		protocol:             &benchProtocol{},
		receiverShutdownChan: make(chan struct{}),
	}
	conn.rwBuffer.Reader = bufio.NewReader(source)

	b.SetBytes(int64(1024 * (4 + len(benchEventJSON))))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		source.Reset(compressed.Bytes())
		conn.rwBuffer.Reader.Reset(source)
		decompressor, err := zlib.NewReader(conn)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := io.Copy(io.Discard, decompressor); err != nil {
			b.Fatal(err)
		}
	}
}
