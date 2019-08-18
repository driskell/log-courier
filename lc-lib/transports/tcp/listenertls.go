package tcp

import (
	"crypto/tls"
	"crypto/x509"
	"net"
)

type listenerTLS struct {
	listenerTCP

	tlsConfig *tls.Config
}

// start configures TLS and begins the accept loop
func (t *listenerTLS) Start(desc string, addr *net.TCPAddr) (bool, error) {
	t.tlsConfig = new(tls.Config)

	// Disable SSLv3 (mitigate POODLE vulnerability)
	t.tlsConfig.MinVersion = tls.VersionTLS10

	// Set the certificate if we set one
	if t.config.certificate != nil {
		t.tlsConfig.Certificates = []tls.Certificate{*t.config.certificate}
	}

	// Set CA for client verification
	t.tlsConfig.ClientCAs = x509.NewCertPool()
	for _, cert := range t.config.caList {
		t.tlsConfig.ClientCAs.AddCert(cert)
	}

	return t.listenerTCP.Start(desc, addr)
}

// newConnection sets up a new connection
func (t *listenerTLS) newConnection(socket net.Conn) {
	log.Notice("[%s] New connection from %s", t.pool.Server(), socket.RemoteAddr().String())

	conn := &connectionTLS{
		connectionTCP: connectionTCP{
			socket:         socket,
			context:        t.context,
			poolServer:     t.pool.Server(),
			eventChan:      t.eventChan,
			connectionChan: t.connectionChan,
			server:         true,
			// TODO: Additional small routine to collapse same-nonce acks into a 10-size channel matching receive limit?
			sendChan: make(chan protocolMessage, 1),
		},
		server:    true,
		tlsConfig: t.tlsConfig,
	}
	t.connectionChan <- &socketMessage{conn: conn}
}
