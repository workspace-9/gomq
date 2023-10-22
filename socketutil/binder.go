package socketutil

import (
  "context"
  "net"

  "github.com/exe-or-death/gomq"
  "github.com/exe-or-death/gomq/zmtp"
  "github.com/exe-or-death/gomq/transport"
)

type IncomingConnectionHandler func(zmtp.Socket) error

type BindDriver struct {
  ctx context.Context
  cancel context.CancelFunc
  transport transport.Transport
  mechanism zmtp.Mechanism
  address string
  handler IncomingConnectionHandler
  eventBus gomq.EventBus
  meta MetadataProvider
  metaHandler MetadataHandler
  done chan struct{}
  final error
}

type BindDriverHandle struct {
  Driver *BindDriver
  Queue chan zmtp.CommandOrMessage
}

func (b *BindDriver) Setup(
  ctx context.Context,
  tp transport.Transport,
  mech zmtp.Mechanism,
  addr string,
  handler IncomingConnectionHandler,
  eventBus gomq.EventBus,
  meta MetadataProvider,
  metaHandler MetadataHandler,
) {
  derived, cancel := context.WithCancel(ctx)
  b.ctx = derived
  b.cancel = cancel
  b.transport = tp
  b.mechanism = mech
  b.address = addr
  b.handler = handler
  b.eventBus = eventBus
  b.meta = meta
  b.metaHandler = metaHandler
  b.done = make(chan struct{})
} 

func (b *BindDriver) run() error {
  listener, err := b.transport.Bind(b.address)
  if err != nil {
    return err
  }

  for {
    conn, err := listener.Accept()
    if err != nil {
      b.eventBus.Post(gomq.Event{
        gomq.EventTypeAcceptFailed,
        transport.BuildURL(conn.LocalAddr(), b.transport),
        "",
        err.Error(),
      })
      return err
    }

    b.eventBus.Post(gomq.Event{
      gomq.EventTypeAccepted,
      transport.BuildURL(conn.LocalAddr(), b.transport),
      transport.BuildURL(conn.RemoteAddr(), b.transport),
      err.Error(),
    })

    go b.handleConn(conn)
  }
}

func (b *BindDriver) handleConn(conn net.Conn) {
  greeting := zmtp.NewGreeting()
  greeting.SetVersionMajor(3)
  greeting.SetVersionMinor(1)
  greeting.SetMechanism(b.mechanism.Name())
  if _, err := greeting.WriteTo(conn); err != nil {
    b.eventBus.Post(gomq.Event{
      gomq.EventTypeFailedGreeting,
      transport.BuildURL(conn.LocalAddr(), b.transport),
      transport.BuildURL(conn.RemoteAddr(), b.transport),
      err.Error(),
    })
    return
  }

  if _, err := greeting.ReadFrom(conn); err != nil {
    b.eventBus.Post(gomq.Event{
      gomq.EventTypeFailedGreeting,
      transport.BuildURL(conn.LocalAddr(), b.transport),
      transport.BuildURL(conn.RemoteAddr(), b.transport),
      err.Error(),
    })
    return
  }

  if err := b.mechanism.ValidateGreeting(&greeting); err != nil {
    b.eventBus.Post(gomq.Event{
      gomq.EventTypeFailedGreeting,
      transport.BuildURL(conn.LocalAddr(), b.transport),
      transport.BuildURL(conn.RemoteAddr(), b.transport),
      err.Error(),
    })
    return 
  }

  sock, meta, err := b.mechanism.Handshake(conn, b.meta())
  if err != nil {
    b.eventBus.Post(gomq.Event{
      gomq.EventTypeFailedHandshake,
      transport.BuildURL(conn.LocalAddr(), b.transport),
      transport.BuildURL(conn.RemoteAddr(), b.transport),
      err.Error(),
    })
    return
  }

  if err := b.metaHandler(meta); err != nil {
    b.eventBus.Post(gomq.Event{
      gomq.EventTypeFailedHandshake,
      transport.BuildURL(conn.LocalAddr(), b.transport),
      transport.BuildURL(conn.RemoteAddr(), b.transport),
      err.Error(),
    })
    return
  }

  b.eventBus.Post(gomq.Event{
    gomq.EventTypeReady,
    transport.BuildURL(conn.LocalAddr(), b.transport),
    transport.BuildURL(conn.RemoteAddr(), b.transport),
    err.Error(),
  })

  b.handler(sock)
  b.eventBus.Post(gomq.Event{
    gomq.EventTypeDisconnected,
    transport.BuildURL(conn.LocalAddr(), b.transport),
    transport.BuildURL(conn.RemoteAddr(), b.transport),
    err.Error(),
  })
}
