package addresspool

import "net"

// Address is an address item in a pool that is a single fixed address
type Address struct {
	host string
	desc string
	addr *net.TCPAddr
}

func (a *Address) Host() string {
	return a.host
}

func (a *Address) Desc() string {
	return a.desc
}

func (a *Address) Addr() *net.TCPAddr {
	return a.addr
}
