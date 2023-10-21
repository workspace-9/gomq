package zmtp

import (
	"net"
)

// Mechanism implements the setup of new Connections.
type Mechanism interface {
	// Name of the mechanism.
	Name() string

	// ValidateGreeting returns an error if the greeting from another side is invalid for this mechanism.
	ValidateGreeting(*Greeting) error

	// Handshake performs a handshake with the Connection.
	Handshake(net.Conn, Metadata) (Socket, Metadata, error)

	// Server field for the greeting for this handshake.
	Server() bool
}

// Socket.
type Socket interface {
	// Read the next part of traffic.
	Read() (CommandOrMessage, error)

	// Send a message on the socket.
	SendMessage(Message) error

	// SendCommand sends a command on the socket.
	SendCommand(Command) error

	// Close the socket.
	Close() error
}

// CommandOrMessage may either contain a command or a message.
type CommandOrMessage struct {
	// IsMessage is true iff this contains a message.
	IsMessage bool

	// Command is non nil iff message is nil.
	Command *Command

	// Message is non nil iff IsMessage is true.
	Message *Message
}