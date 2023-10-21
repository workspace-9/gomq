package tcp

import (
  "context"
  "net"
)

// TCPTransport implements transport.Transport
type TCPTransport struct{}

// Bind to a tcp address.
func (TCPTransport) Bind(addr string) (net.Listener, error) {
  tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
  if err != nil {
    return nil, err
  }

  return net.ListenTCP("tcp", tcpAddr)
}

// Connect to a tcp address.
func (TCPTransport) Connect(ctx context.Context, addr string) (net.Conn, error) {
  var d net.Dialer
  return d.DialContext(ctx, "tcp", addr)
}
