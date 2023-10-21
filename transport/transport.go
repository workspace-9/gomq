package transport

import (
	"context"
	"net"
)

// Transport represents a method of generating sockets.
type Transport interface {
	// Bind creates an Listener if successful.
	Bind(addr string) (net.Listener, error)

	// Connect to a remote address.
	Connect(
		ctx context.Context,
		addr string,
	) (net.Conn, error)
}
