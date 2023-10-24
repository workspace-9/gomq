package push

import (
  "context"
  "fmt"

  "github.com/exe-or-death/gomq"
  "github.com/exe-or-death/gomq/types"
  "github.com/exe-or-death/gomq/socketutil"
  "github.com/exe-or-death/gomq/transport"
  "github.com/exe-or-death/gomq/zmtp"
)

// Push implements the zmq push socket.
type Push struct {
  context.Context
  Cancel context.CancelFunc
  *gomq.Config
  Mech zmtp.Mechanism
  ConnectionDrivers map[string]*socketutil.ConnectionDriver
  BindDrivers map[string]*socketutil.BindDriver
  ConnectionHandles map[string]socketutil.WaitCloser[struct{}]
  EventBus gomq.EventBus
  WritePoint chan []zmtp.Message
}

func (p *Push) Name() string {
  return "PUSH"
}

func (p *Push) Connect(tp transport.Transport, addr string) error {
  if _, ok := p.ConnectionHandles[addr]; ok {
    return fmt.Errorf("%w: %s", types.ErrAlreadyConnected, addr)
  }

  var queue chan zmtp.Message
  driver := &socketutil.ConnectionDriver{}
  driver.Setup(
    p.Context,
    p.Mech,
    tp,
    addr,
    p.Config,
    p.EventBus,
    func(ctx context.Context, s zmtp.Socket) error {
      return HandleSock(ctx, s, queue)
    },
    p.Meta,
    p.MetaHandler,
  )
  fatal, err := driver.TryConnect()
  if err != nil && fatal {
    return err
  }
  queue = make(chan zmtp.Message, p.Config.QueueLen())
  wc := socketutil.NewWaitCloser[struct{}](p.Context)
  go PullFromWritePoint(&wc, queue, p.WritePoint)
  p.ConnectionDrivers[addr] = driver
  p.ConnectionHandles[addr] = wc
  go driver.Run()
  return nil
}

func (p *Push) Bind(tp transport.Transport, addr string) error {
  driver := &socketutil.BindDriver{}
  driver.Setup(
    p.Context,
    tp,
    p.Mech,
    addr,
    func(ctx context.Context, s zmtp.Socket) error {
      queue := make(chan zmtp.Message, p.Config.QueueLen())
      wc := socketutil.NewWaitCloser[struct{}](p.Context)
      defer wc.Finish(struct{}{})
      go PullFromWritePoint(&wc, queue, p.WritePoint)
      return HandleSock(ctx, s, queue)
    },
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

func PullFromWritePoint(wc *socketutil.WaitCloser[struct{}], push chan<- zmtp.Message, writePoint chan []zmtp.Message) {
  for {
    select {
    case message := <- writePoint:
      for _, part := range message {
        select {
        case push <- part:
        case <- wc.Done():
          return
        }
      }
    case <-wc.Done():
      return
    }
  }
}

func HandleSock(ctx context.Context, sock zmtp.Socket, queue <-chan zmtp.Message) (err error) {
  for {
    select {
    case msg := <-queue:
      if err := sock.SendMessage(msg); err != nil {
        return err
      }
    case <-ctx.Done():
      return ctx.Err()
    }
  }
}

func (p *Push) Meta() zmtp.Metadata {
  meta := zmtp.Metadata{}
  meta.AddProperty("Socket-Type", "PUSH")
  return meta
}

func (p *Push) MetaHandler(meta zmtp.Metadata) error {
  var err error
  meta.Properties(func(name string, value string) {
    if name == "Socket-Type" && err == nil {
      if value != "PULL" {
        err = fmt.Errorf("Expected pull socket to connect, got %s", value)
      }
    }
  })

  return err
}

func (p *Push) Send(data []zmtp.Message) error {
  select {
  case p.WritePoint <- data:
    return nil
  case <-p.Context.Done():
    return p.Context.Err()
  }
}

func (p *Push) Recv() ([]zmtp.Message, error) {
  return nil, types.ErrOperationNotPermitted
}

func (p *Push) Close() error {
  p.Cancel()
  for _, conn := range p.ConnectionDrivers {
    conn.Close()
  }
  for _, conn := range p.BindDrivers {
    conn.Close()
  }
  return nil
}
