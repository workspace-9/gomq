package gomq

// driver is responsible for implementing socket operations.
type driver interface {
  sendMessage([][]byte) error
  recvMessage() ([][]byte, error)
  typ() SocketType
}
