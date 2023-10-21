package zmtp

import (
  "io"
)

// Handshaker performs a zmtp handshake.
type Handshaker interface {
  // Handshake runs the handshake.
  Handshake(io.ReadWriter) error
}
