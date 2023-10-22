package types

import (
  "github.com/exe-or-death/gomq"
  "github.com/exe-or-death/gomq/transport"
  "github.com/exe-or-death/gomq/zmtp"
)

// SocketType represents a type of socket in a communitcation pattern.
type SocketType interface {
  // Name of the type.
  Name() string

  // Connect to the remote address using the given transport.
  Connect(tp transport.Transport, addr string) error

  // Bind to the given address using the given transport.
  Bind(tp transport.Transport, addr string) error
  
  // Send a message over the socket.
  Send(zmtp.Message) error

  // Recv either a command or a message on the socket.
  Recv() (zmtp.CommandOrMessage, error)
}

// SocketConstructor constructs a socket.
type SocketConstructor interface {
  // ConstructSocket constructs a new socket using the given
  // mechanism.
  ConstructSocket(mech zmtp.Mechanism, conf gomq.Config, eventBus chan<-gomq.Event) error
}
