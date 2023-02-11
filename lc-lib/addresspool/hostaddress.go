package addresspool

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

type hostAddress struct {
	expire    time.Time
	desc      string
	hostPort  string
	addresses []*Address
}

func (h *hostAddress) populateAddresses() error {
	// Standard host:port declaration
	var host string
	var portStr string
	var port uint64
	var err error
	if host, portStr, err = net.SplitHostPort(h.hostPort); err != nil {
		return fmt.Errorf("invalid hostport given: %s", h.hostPort)
	}

	if port, err = strconv.ParseUint(portStr, 10, 16); err != nil {
		return fmt.Errorf("invalid port given: %s", portStr)
	}

	if ip := net.ParseIP(host); ip != nil {
		// IP address
		h.addresses = append(h.addresses, &Address{
			host: host,
			desc: h.desc,
			addr: &net.TCPAddr{
				IP:   ip,
				Port: int(port),
			},
		})
		return nil
	}

	// Lookup the hostname in DNS
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("DNS lookup failure \"%s\": %s", host, err)
	} else if len(ips) == 0 {
		return fmt.Errorf("DNS lookup failure \"%s\": No addresses found", host)
	}

	for _, ip := range ips {
		h.addresses = append(h.addresses, &Address{
			host: host,
			desc: fmt.Sprintf("%s (%s:%d)", h.desc, ip, port),
			addr: &net.TCPAddr{
				IP:   ip,
				Port: int(port),
			},
		})
	}

	return nil
}

func (h *hostAddress) Next() (*Address, error) {
	if h.addresses == nil {
		if err := h.populateAddresses(); err != nil {
			return nil, err
		}
	}

	if len(h.addresses) == 0 {
		return nil, nil
	}

	address := h.addresses[0]
	h.addresses = h.addresses[1:]
	return address, nil
}

func (h *hostAddress) Finished() bool {
	if h.expire.Before(time.Now()) {
		return true
	}
	return h.addresses != nil && len(h.addresses) == 0
}
