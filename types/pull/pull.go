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
  ConnectionHandles map[string]socketutil.WaitCloser[struct{}]
  BindDrivers map[string]*socketutil.BindDriver
  ReadPoint chan []zmtp.Message
  EventBus gomq.EventBus
}

func (p *Pull) Name() string {
  return "PULL"
}

func (p *Pull) Connect(tp transport.Transport, addr string) error {
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
  go PushIntoReadPoint(&wc, queue, p.ReadPoint)
  p.ConnectionDrivers[addr] = driver
  p.ConnectionHandles[addr] = wc
  go driver.Run()
  return nil
}

func (p *Pull) Bind(tp transport.Transport, addr string) error {
  if _, ok := p.BindDrivers[addr]; ok {
    return fmt.Errorf("%w: %s", types.ErrAlreadyBound, addr)
  }

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
      go PushIntoReadPoint(&wc, queue, p.ReadPoint)
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

func PushIntoReadPoint(wc *socketutil.WaitCloser[struct{}], pull <-chan zmtp.Message, readPoint chan []zmtp.Message) {
  defer wc.Finish(struct{}{})
  built := make([]zmtp.Message, 0)
  for {
    select {
    case part := <-pull:
      built = append(built, part)
      if !part.More {
        select {
        case readPoint <- built:
          built = make([]zmtp.Message, 0)
        case <-wc.Done():
          return
        }
      }
    case <-wc.Done():
      return
    }
  }
}

func HandleSock(ctx context.Context, sock zmtp.Socket, queue chan<- zmtp.Message) (err error) {
  for {
    next, err := sock.Read()
    if err != nil {
      return err
    }

    if !next.IsMessage {
      continue
    }

    if !next.Message.More {
      select {
      case queue <- *next.Message:
      case <- ctx.Done():
      }
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
  case msg := <- p.ReadPoint:
    return msg, nil
  case <-p.Context.Done():
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
