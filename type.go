package gomq

type SocketType string

const (
  SocketTypePush = SocketType("PUSH")
  SocketTypePull = SocketType("PULL")
  SocketTypeReq = SocketType("REQ")
  SocketTypeResp = SocketType("RESP")
  SocketTypePair = SocketType("PAIR")
  SocketTypePub = SocketType("PUB")
  SocketTypeSub = SocketType("SUB")
  SocketTypeXPub = SocketType("XPUB")
  SocketTypeXSub = SocketType("XSUB")
)
