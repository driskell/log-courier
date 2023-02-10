package admin

import "testing"

func TestAdminConnectAddressOnlyNoPort(t *testing.T) {
	connect := splitAdminConnectString("127.0.0.1")

	if connect[0] != "tcp" {
		t.Errorf("Expected connection type tcp, received: %s", connect[0])
	}
	if connect[1] != "127.0.0.1" {
		t.Errorf("Expected connection address 127.0.0.1, received: %s", connect[1])
	}
}

func TestAdminConnectAddressOnlyWithPort(t *testing.T) {
	connect := splitAdminConnectString("127.0.0.1:1234")

	if connect[0] != "tcp" {
		t.Errorf("Expected connection type tcp, received: %s", connect[0])
	}
	if connect[1] != "127.0.0.1:1234" {
		t.Errorf("Expected connection address 127.0.0.1:1234, received: %s", connect[1])
	}
}

func TestAdminConnectAddressEmptyTransport(t *testing.T) {
	connect := splitAdminConnectString(":127.0.0.1:1234")

	if connect[0] != "tcp" {
		t.Errorf("Expected connection type tcp, received: %s", connect[0])
	}
	if connect[1] != "127.0.0.1:1234" {
		t.Errorf("Expected connection address 127.0.0.1:1234, received: %s", connect[1])
	}
}

func TestAdminConnectTcpNoPort(t *testing.T) {
	connect := splitAdminConnectString("tcp:127.0.0.1")

	if connect[0] != "tcp" {
		t.Errorf("Expected connection type tcp, received: %s", connect[0])
	}
	if connect[1] != "127.0.0.1" {
		t.Errorf("Expected connection address 127.0.0.1, received: %s", connect[1])
	}
}

func TestAdminConnectTcpWithPort(t *testing.T) {
	connect := splitAdminConnectString("tcp:127.0.0.1:1234")

	if connect[0] != "tcp" {
		t.Errorf("Expected connection type tcp, received: %s", connect[0])
	}
	if connect[1] != "127.0.0.1:1234" {
		t.Errorf("Expected connection address 127.0.0.1:1234, received: %s", connect[1])
	}
}

func TestAdminConnectTcp6NoPort(t *testing.T) {
	connect := splitAdminConnectString("tcp:[::1]")

	if connect[0] != "tcp" {
		t.Errorf("Expected connection type tcp, received: %s", connect[0])
	}
	if connect[1] != "[::1]" {
		t.Errorf("Expected connection address [::1], received: %s", connect[1])
	}
}

func TestAdminConnectTcp6WithPort(t *testing.T) {
	connect := splitAdminConnectString("tcp:[::1]:1234")

	if connect[0] != "tcp" {
		t.Errorf("Expected connection type tcp, received: %s", connect[0])
	}
	if connect[1] != "[::1]:1234" {
		t.Errorf("Expected connection address [::1]:1234, received: %s", connect[1])
	}
}

func TestAdminConnectUnix(t *testing.T) {
	connect := splitAdminConnectString("unix:/var/path/somewhere")

	if connect[0] != "unix" {
		t.Errorf("Expected connection type unix, received: %s", connect[0])
	}
	if connect[1] != "/var/path/somewhere" {
		t.Errorf("Expected connection address /var/path/somewhere, received: %s", connect[1])
	}
}

func TestAdminConnectUnixAlone(t *testing.T) {
	connect := splitAdminConnectString("unix")

	if connect[0] != "unix" {
		t.Errorf("Expected connection type unix, received: %s", connect[0])
	}
	if connect[1] != "" {
		t.Errorf("Expected empty connection address, received: %s", connect[1])
	}
}

func TestAdminConnectTcpAlone(t *testing.T) {
	connect := splitAdminConnectString("tcp")

	if connect[0] != "tcp" {
		t.Errorf("Expected connection type tcp, received: %s", connect[0])
	}
	if connect[1] != "" {
		t.Errorf("Expected empty connection address, received: %s", connect[1])
	}
}
