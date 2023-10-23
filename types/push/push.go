package push

import (
  "sync"
  "context"
  "fmt"

  "github.com/exe-or-death/gomq"
  "github.com/exe-or-death/gomq/types"
  "github.com/exe-or-death/gomq/socketutil"
  "github.com/exe-or-death/gomq/transport"
  "github.com/exe-or-death/gomq/zmtp"
  wk8 "github.com/wk8/go-ordered-map/v2"
)

// Push implements the zmq push socket.
type Push struct {
  sync.RWMutex
  context.Context
  Cancel context.CancelFunc
  *gomq.Config
  Mech zmtp.Mechanism
  ConnectionDrivers map[string]*socketutil.ConnectionDriver
  BindDrivers map[string]*socketutil.BindDriver
  Queues struct {
    *wk8.OrderedMap[int64, chan[]zmtp.Message]
    Idx int64
  }
  EventBus gomq.EventBus
  Buffer chan []zmtp.Message
  LastPush *wk8.Pair[int64, chan[]zmtp.Message]
}

func (p *Push) Name() string {
  return "PUSH"
}

func (p *Push) Connect(tp transport.Transport, addr string) error {
  p.Lock()
  defer p.Unlock()
  var queue chan []zmtp.Message
  driver := &socketutil.ConnectionDriver{}
  driver.Setup(
    p.Context,
    p.Mech,
    tp,
    addr,
    p.Config,
    p.EventBus,
    func(s zmtp.Socket) (fatal bool, err error) {
      return p.HandleSock(s, queue)
    },
    p.Meta,
    p.MetaHandler,
  )
  fatal, err := driver.TryConnect()
  if err != nil && fatal {
    return err
  }
  var idx int64
  queue, idx = p.NewConnQueue()
  p.ConnectionDrivers[addr] = driver
  go func() {
    driver.Run()
    driver.Close()
    p.Lock()
    delete(p.ConnectionDrivers, addr)
    p.Queues.Delete(idx)
    p.Unlock()
  }()

  return nil
}

func (p *Push) Bind(tp transport.Transport, addr string) error {
  p.Lock()
  defer p.Unlock()
  driver := &socketutil.BindDriver{}
  driver.Setup(
    p.Context,
    tp,
    p.Mech,
    addr,
    func(s zmtp.Socket) (fatal bool, err error) {
      p.Lock()
      queue, idx := p.NewConnQueue()
      p.Unlock()

      defer func() {
        p.Lock()
        p.Queues.Delete(idx)
        p.Unlock()
      }()

      return p.HandleSock(s, queue)
    },
    p.EventBus,
    p.Meta,
    p.MetaHandler,
  )
  if err := driver.TryBind(); err != nil {
    return err
  }
  p.BindDrivers[addr] = driver
  go func() {
    driver.Run()
    driver.Close()
    p.Lock()
    delete(p.BindDrivers, addr)
    p.Unlock()
  }()
  return nil
}

func (p *Push) NewConnQueue() (queue chan []zmtp.Message, idx int64) {
  idx = p.Queues.Idx
  p.Queues.Idx++
  queue = make(chan []zmtp.Message)
  p.Queues.Set(idx, queue)
  return queue, idx
}

func (p *Push) HandleSock(sock zmtp.Socket, queue chan[]zmtp.Message) (fatal bool, err error) {
  for message := range queue {
    for idx, datum := range message {
      if err := sock.SendMessage(zmtp.Message{
        More: idx != len(message) - 1, Body: datum.Body,
      }); err != nil {
        return false, err
      }

    }
  }

  return false, nil
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
  // We only need an rlock because we aren't manipulating the maps.
  // The mutex only protects the map, nothing else is guaranteed by the
  // spec as the socket itself need not be thread safe.
  p.RLock()
  defer p.RUnlock()

  if p.Queues.Len() == 0 {
    return ErrNoPeers
  }

  // Try to push to each queue in a round robin fashion.
  start := p.LastPush
  var cur *wk8.Pair[int64, chan[]zmtp.Message]
  if start == nil {
    cur = p.Queues.Oldest()
  } else {
    cur = start.Next()
  }
  for cur != start {
    if cur == nil {
      cur = p.Queues.Oldest()
    }

    select {
    case cur.Value <- data:
      p.LastPush = cur
      return nil
    default:
    }
  }

  return ErrAllPeersBusy
}

type allPeersBusy struct {}

func (allPeersBusy) Error() string {
  return "All peers busy"
}

var ErrAllPeersBusy allPeersBusy

type noPeers struct {}

func (noPeers) Error() string {
  return "No peers"
}

var ErrNoPeers noPeers

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
