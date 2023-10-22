package tcp

import (
	"context"
	"net"
)

// TCPTransport implements transport.Transport
type TCPTransport struct{}

// Name of the transport is tcp.
func (TCPTransport) Name() string {
  return "tcp"
}

// Bind to a tcp address.
func (TCPTransport) Bind(addr string) (net.Listener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	return net.ListenTCP("tcp", tcpAddr)
}

// Connect to a tcp address.
func (TCPTransport) Connect(
  ctx context.Context, 
  addr string,
) (
  conn net.Conn, 
  fatal bool, 
  err error,
) {
  _, err = net.ResolveTCPAddr("tcp", addr)
  if err != nil {
    return nil, true, err
  }

	var d net.Dialer
  conn, err = d.DialContext(ctx, "tcp", addr)
  return conn, false, err
}
