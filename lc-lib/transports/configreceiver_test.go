package transports

import "testing"

func TestReceiverConfigAcceptsDeprecatedMaxPendingPayloads(t *testing.T) {
	entry := &ReceiverConfigEntry{Listen: []string{"127.0.0.1:1234"}, MaxPendingPayloads: 10}
	if err := entry.Validate(nil, "/receivers[0]/"); err != nil {
		t.Fatalf("Expected the deprecated max pending payloads to be accepted, got %s", err)
	}
}
