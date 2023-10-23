package push

import (
  "context"

  "github.com/exe-or-death/gomq"
  "github.com/exe-or-death/gomq/socketutil"
  "github.com/exe-or-death/gomq/zmtp"
  wk8 "github.com/wk8/go-ordered-map/v2"
)

func init() {
  gomq.RegisterSocketType(
    "PUSH",
    func(ctx context.Context, mech zmtp.Mechanism, conf *gomq.Config, eventBus gomq.EventBus) (gomq.SocketDriver, error) {
      derived, cancel := context.WithCancel(ctx)
      sock := &Push{
        Context: derived,
        Cancel: cancel,
        Config: conf,
        Mech: mech,
        ConnectionDrivers: map[string]*socketutil.ConnectionDriver{},
        BindDrivers: map[string]*socketutil.BindDriver{},
        EventBus: eventBus,
        Buffer: make(chan []zmtp.Message, conf.QueueLen()),
      }
      sock.Queues.OrderedMap = wk8.New[int64, chan[]zmtp.Message]()

      return sock, nil
    },
  )
}
