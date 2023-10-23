package socketutil

import (
  "context"
  "time"

  "github.com/exe-or-death/gomq"
  "github.com/exe-or-death/gomq/transport"
  "github.com/exe-or-death/gomq/zmtp"
)

type SocketHandler func(zmtp.Socket) (fatal bool, err error)

type MetadataProvider func() zmtp.Metadata

type MetadataHandler func(zmtp.Metadata) error

type ConnectionDriver struct {
  ctx context.Context
  mechanism zmtp.Mechanism
  socket zmtp.Socket
  transport transport.Transport
  address string
  config *gomq.Config
  eventBus gomq.EventBus
  handler SocketHandler
  meta MetadataProvider
  metaHandler MetadataHandler
  cancelFunc context.CancelFunc
  done chan struct{}
  final error
}

type ConnectionDriverHandle struct {
  Driver *ConnectionDriver
  Queue chan zmtp.CommandOrMessage
}

func (c *ConnectionDriver) Close() error {
  c.cancelFunc()
  if c.socket != nil {
    c.socket.Close()
  }
  <-c.done
  return c.final
}

func (c *ConnectionDriver) TryConnect() (fatal bool, err error) {
  ctx, cancel := context.WithTimeout(c.ctx, c.config.ConnectTimeout())
  conn, fatal, err := c.transport.Connect(ctx, c.address)
  cancel()
  if err != nil {
    c.eventBus.Post(gomq.Event{
      gomq.EventTypeConnectFailed,
      "",
      c.address,
      err.Error(),
    })
    return fatal, err
  }
  c.eventBus.Post(gomq.Event{
    gomq.EventTypeConnected,
    transport.BuildURL(conn.LocalAddr(), c.transport),
    transport.BuildURL(conn.RemoteAddr(), c.transport),
    "",
  })

  greeting := zmtp.NewGreeting()
  greeting.SetVersionMajor(3)
  greeting.SetVersionMinor(1)
  greeting.SetMechanism(c.mechanism.Name())
  if _, err := greeting.WriteTo(conn); err != nil {
    c.eventBus.Post(gomq.Event{
      gomq.EventTypeFailedGreeting,
      transport.BuildURL(conn.LocalAddr(), c.transport),
      transport.BuildURL(conn.RemoteAddr(), c.transport),
      err.Error(),
    })
    return false, err
  }

  if _, err := greeting.ReadFrom(conn); err != nil {
    c.eventBus.Post(gomq.Event{
      gomq.EventTypeFailedGreeting,
      transport.BuildURL(conn.LocalAddr(), c.transport),
      transport.BuildURL(conn.RemoteAddr(), c.transport),
      err.Error(),
    })
    return false, err
  }

  if err := c.mechanism.ValidateGreeting(&greeting); err != nil {
    c.eventBus.Post(gomq.Event{
      gomq.EventTypeFailedGreeting,
      transport.BuildURL(conn.LocalAddr(), c.transport),
      transport.BuildURL(conn.RemoteAddr(), c.transport),
      err.Error(),
    })
    return false, err
  }

  sock, meta, err := c.mechanism.Handshake(conn, c.meta())
  if err != nil {
    c.eventBus.Post(gomq.Event{
      gomq.EventTypeFailedHandshake,
      transport.BuildURL(conn.LocalAddr(), c.transport),
      transport.BuildURL(conn.RemoteAddr(), c.transport),
      err.Error(),
    })
    return false, err
  }

  if err := c.metaHandler(meta); err != nil {
    c.eventBus.Post(gomq.Event{
      gomq.EventTypeFailedHandshake,
      transport.BuildURL(conn.LocalAddr(), c.transport),
      transport.BuildURL(conn.RemoteAddr(), c.transport),
      err.Error(),
    })
    return false, err
  }

  c.socket = sock
  c.eventBus.Post(gomq.Event{
    gomq.EventTypeReady,
    transport.BuildURL(conn.LocalAddr(), c.transport),
    transport.BuildURL(conn.RemoteAddr(), c.transport),
    "",
  })
  return false, nil
}

func (c *ConnectionDriver) Setup(
  ctx context.Context,
  mech zmtp.Mechanism,
  tp transport.Transport,
  addr string,
  conf *gomq.Config,
  eventBus gomq.EventBus,
  handler SocketHandler,
  meta MetadataProvider,
  metaHandler MetadataHandler,
) {
  derived, cancel := context.WithCancel(ctx)
  c.ctx = derived
  c.cancelFunc = cancel
  c.mechanism = mech
  c.transport = tp
  c.address = addr
  c.config = conf
  c.eventBus = eventBus
  c.handler = handler
  c.meta = meta
  c.metaHandler = metaHandler
  c.done = make(chan struct{})
}

func (c *ConnectionDriver) Run() {
  c.final = c.run()
  c.cancelFunc()
  close(c.done)
}

func (c *ConnectionDriver) run() error {
  for {
    if err := c.ctx.Err(); err != nil {
      return c.ctx.Err()
    }

    if c.socket == nil {
      connectStart := time.Now()
      fatal, err := c.TryConnect()
      if err != nil {
        if fatal {
          return err
        }

        time.Sleep(c.config.ReconnectTimeout() - time.Since(connectStart))
        continue
      }
    }

    fatal, err := c.handler(c.socket)
    if err != nil {
      c.eventBus.Post(gomq.Event{
        gomq.EventTypeDisconnected, 
        transport.BuildURL(c.socket.Net().LocalAddr(), c.transport),
        transport.BuildURL(c.socket.Net().RemoteAddr(), c.transport),
        err.Error(),
      })

      if fatal {
        return err
      }
      time.Sleep(c.config.ReconnectTimeout())
      c.socket = nil
    }
  }
}
