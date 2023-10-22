package pull

import (
  "context"
  "fmt"

  "github.com/exe-or-death/gomq"
  "github.com/exe-or-death/gomq/socketutil"
  "github.com/exe-or-death/gomq/transport"
  "github.com/exe-or-death/gomq/zmtp"
)

// Pull implements the zmq pull socket.
type Pull struct {
  context.Context
  gomq.Config
  Mech zmtp.Mechanism
  ConnectionDrivers map[string]*socketutil.ConnectionDriver
  BindDrivers map[string]*socketutil.BindDriver
  EventBus gomq.EventBus
  Buffer chan []zmtp.Message
}

func (p *Pull) Connect(tp transport.Transport, addr string) error {
  driver := &socketutil.ConnectionDriver{}
  driver.Setup(
    p.Context,
    p.Mech,
    tp,
    addr,
    &p.Config,
    p.EventBus,
    p.HandleSock,
    p.Meta,
    p.MetaHandler,
  )
  fatal, err := driver.TryConnect()
  if err != nil && fatal {
    return err
  }
  go driver.Run()
  p.ConnectionDrivers[addr] = driver

  return nil
}

func (p *Pull) HandleSock(sock zmtp.Socket) (fatal bool, err error) {
  builtMessage := make([]zmtp.Message, 0)
  for {
    next, err := sock.Read()
    if err != nil {
      return false, err
    }

    if !next.IsMessage {
      continue
    }

    builtMessage = append(builtMessage, *next.Message)
    if !next.Message.More {
      p.Buffer <- builtMessage
      builtMessage = make([]zmtp.Message, 0)
    }
  }
}

func (p *Pull) Meta() zmtp.Metadata {
  meta := zmtp.Metadata{}
  meta.AddProperty("Socket-Type", "PULL")
  return meta
}

func (p *Pull) MetaHandler(meta zmtp.Metadata) error {
  var err error
  meta.Properties(func(name string, value string) {
    if name == "Socket-Type" && err == nil {
      if value != "PUSH" {
        err = fmt.Errorf("Expected push socket to connect, got %s", value)
      }
    }
  })

  return err
}
