package tcp

import (
	"context"
	"net"
	"net/url"

	"github.com/exe-or-death/gomq"
	"github.com/exe-or-death/gomq/transport"
)

func init() {
	gomq.RegisterTransport("tcp", func() transport.Transport {
		return TCPTransport{}
	})
}

// TCPTransport implements transport.Transport
type TCPTransport struct{}

// Name of the transport is tcp.
func (TCPTransport) Name() string {
	return "tcp"
}

// Bind to a tcp address.
func (TCPTransport) Bind(url *url.URL) (net.Listener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", url.Host)
	if err != nil {
		return nil, err
	}

	return net.ListenTCP("tcp", tcpAddr)
}

// Connect to a tcp address.
func (TCPTransport) Connect(
	ctx context.Context,
	url *url.URL,
) (
	conn net.Conn,
	fatal bool,
	err error,
) {
	_, err = net.ResolveTCPAddr("tcp", url.Host)
	if err != nil {
		return nil, true, err
	}

	var d net.Dialer
	conn, err = d.DialContext(ctx, "tcp", url.Host)
	return conn, false, err
}
