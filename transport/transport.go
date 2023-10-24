package transport

import (
	"context"
  "fmt"
	"net"
	"net/url"
)

// Transport represents a method of generating sockets.
type Transport interface {
  // Name of the transport.
  Name() string

	// Bind creates an Listener if successful.
	Bind(url *url.URL) (net.Listener, error)

	// Connect to a remote address.
	Connect(
		ctx context.Context,
    url *url.URL,
	) (conn net.Conn, fatal bool, err error)
}

// BuildURL builds a URL given a tranposrt and an address.
func BuildURL(addr net.Addr, tp Transport) string {
  return fmt.Sprintf("%s://%s", tp.Name(), addr.String())
}
