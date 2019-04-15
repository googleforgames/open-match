package port

import (
	"fmt"
	"net"

	"github.com/pkg/errors"
)

// Port is a holder for a TCP port.
type Port struct {
	number   int
	listener net.Listener
}

// Obtain returns the TCP listener. This method can only be called once.
func (p *Port) Obtain() (net.Listener, error) {
	listener := p.listener
	p.listener = nil
	if listener == nil {
		return nil, errors.WithStack(fmt.Errorf("cannot Obtain() listener for %d because already handed off", p.number))
	}
	return listener, nil
}

// String returns the port number as a string.
/*
func (p *Port) String() string {
	return fmt.Sprintf("%d", p.number)
}
*/
// Number returns the port number.
func (p *Port) Number() int {
	return p.number
}

// Close shutsdown the TCP listener.
func (p *Port) Close() error {
	if p.listener != nil {
		return p.listener.Close()
	}
	return nil
}

// PortFromNumber opens a TCP based on the port number provided.
func PortFromNumber(port int) (*Port, error) {
	addr := "0.0.0.0"
	conn, err := net.Listen("tcp", addr)
	if err == nil {
		fmt.Printf("SELECTPORT %d\n", port)
		return &Port{
			number:   port,
			listener: conn,
		}, nil
	}
	return nil, err
}

// CreatePort creates a Port object from TCP listener.
func CreatePort(port int, listener net.Listener) *Port {
	return &Port{
		number:   port,
		listener: listener,
	}
}
