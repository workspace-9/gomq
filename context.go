package gomq

import (
  "context"
  "sync"

  "github.com/exe-or-death/gomq/transport"
)

type Context struct {
  sync.RWMutex
  transports map[string]transport.Transport
  ctx context.Context
}

func NewContext(ctx context.Context) *Context {
  return &Context{
    ctx: ctx,
    transports: make(map[string]transport.Transport),
  }
}

func (c *Context) getTransport(name string) (transport.Transport, bool) {
  c.Lock()
  defer c.Unlock()

  if tp, ok := c.transports[name]; ok {
    return tp, ok
  }

  registeredTransports.RLock()
  defer registeredTransports.RUnlock()
  if fac, ok := registeredTransports.transports[name]; ok {
    tp := fac()
    c.transports[name] = tp
    return tp, true
  }

  return nil, false
}

func (c *Context) NewSocket() {

}
