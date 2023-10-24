package gomq

type EventType int

const (
	EventTypeConnected       = EventType(0)
	EventTypeDisconnected    = EventType(1)
	EventTypeConnectFailed   = EventType(2)
	EventTypeAccepted        = EventType(3)
	EventTypeAcceptFailed    = EventType(4)
	EventTypeFailedGreeting  = EventType(5)
	EventTypeFailedHandshake = EventType(6)
	EventTypeReady           = EventType(7)
)

func (e EventType) String() string {
	switch e {
	case EventTypeConnected:
		return "Connected"
	case EventTypeDisconnected:
		return "Disconnected"
	case EventTypeConnectFailed:
		return "Connect failed"
	case EventTypeAccepted:
		return "Accepted"
	case EventTypeAcceptFailed:
		return "Accept failed"
	case EventTypeFailedGreeting:
		return "Failed greeting"
	case EventTypeFailedHandshake:
		return "Failed handshake"
	case EventTypeReady:
		return "Ready"
	}

	return ""
}

type Event struct {
	EventType
	LocalAddr  string
	RemoteAddr string
	Notes      string
}

type EventBus interface {
	Post(Event)
}
