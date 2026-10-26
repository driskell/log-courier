package pipelinebench

import (
	"io"
	"os"
	"testing"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/transports"
	"gopkg.in/op/go-logging.v1"
)

const benchTransport = "pipelinebench"

func TestMain(m *testing.M) {
	backend := logging.AddModuleLevel(logging.NewLogBackend(io.Discard, "", 0))
	backend.SetLevel(logging.ERROR, "")
	logging.SetBackend(backend)

	transports.RegisterReceiver(benchTransport, func(*config.Parser, string, map[string]interface{}, string) (transports.ReceiverFactory, error) {
		return &benchReceiverFactory{}, nil
	})
	transports.RegisterTransport(benchTransport, newBenchTransportFactory)

	os.Exit(m.Run())
}
