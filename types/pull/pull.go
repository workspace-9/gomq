package pull

import (
  "context"
  "fmt"

  "github.com/exe-or-death/gomq"
  "github.com/exe-or-death/gomq/types"
  "github.com/exe-or-death/gomq/socketutil"
  "github.com/exe-or-death/gomq/transport"
  "github.com/exe-or-death/gomq/zmtp"
)

// Pull implements the zmq pull socket.
type Pull struct {
  context.Context
  Cancel context.CancelFunc
  *gomq.Config
  Mech zmtp.Mechanism
  ConnectionDrivers map[string]*socketutil.ConnectionDriver
  BindDrivers map[string]*socketutil.BindDriver
  EventBus gomq.EventBus
  Buffer chan []zmtp.Message
}

func (p *Pull) Name() string {
  return "PULL"
}

func (p *Pull) Connect(tp transport.Transport, addr string) error {
  driver := &socketutil.ConnectionDriver{}
  driver.Setup(
    p.Context,
    p.Mech,
    tp,
    addr,
    p.Config,
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

func (p *Pull) Bind(tp transport.Transport, addr string) error {
  driver := &socketutil.BindDriver{}
  driver.Setup(
    p.Context,
    tp,
    p.Mech,
    addr,
    p.HandleSock,
    p.EventBus,
    p.Meta,
    p.MetaHandler,
  )
  if err := driver.TryBind(); err != nil {
    return err
  }
  p.BindDrivers[addr] = driver
  go driver.Run()
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

func (p *Pull) Send([]zmtp.Message) error {
  return types.ErrOperationNotPermitted
}

func (p *Pull) Recv() ([]zmtp.Message, error) {
  select {
  case data := <-p.Buffer:
    return data, nil
  case <- p.Context.Done():
    return nil, p.Context.Err()
  }
}

func (p *Pull) Close() error {
  p.Cancel()
  for _, conn := range p.ConnectionDrivers {
    conn.Close()
  }
  for _, bind := range p.BindDrivers {
    bind.Close()
  }
  return nil
}
