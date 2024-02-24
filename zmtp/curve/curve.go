package curve

import (
	"fmt"
	"net"

	"github.com/workspace-9/gomq/zmtp"
  "golang.org/x/crypto/nacl/box"
)

type Curve struct {
  serv *CurveServer
  cli *CurveClient
}

type CurveServer struct {
  pubKey, privKey [32]byte
}

type CurveClient struct {
  pubKey, privKey, serverPubKey [32]byte
}

func (c *Curve) Name() string {
	return MechName
}

func (c *Curve) Server() bool {
  return c.serv != nil
}

func (c *Curve) ValidateGreeting(g *zmtp.Greeting) error {
	if g.Server() == c.Server() {
    if g.Server() {
      return ErrBothServers
    }
    return ErrBothClients
	}

	return nil
}

type bothClients struct{}

func (bothClients) Error() string {
	return "Cannot connect two curve clients"
}

var ErrBothClients bothClients

type bothServers struct{}

func (bothServers) Error() string {
	return "Cannot connect two curve servers"
}

var ErrBothServers bothServers

// Handshake performs a null mechanism handshake.
func (c *Curve) Handshake(conn net.Conn, meta zmtp.Metadata) (
	zmtp.Socket,
	zmtp.Metadata,
	error,
) {
  if serv := c.serv; serv != nil {
    return serv.Handshake(conn, meta)
  }

  if c.cli == nil {
    c.cli = &CurveClient{}
  }

  return c.cli.Handshake(conn, meta)
}

func (c *CurveServer) Handshake(conn net.Conn, meta zmtp.Metadata) (
  zmtp.Socket,
  zmtp.Metadata,
  error,
) {
	var cmd zmtp.Command
	if _, err := cmd.ReadFrom(conn); err != nil {
		return nil, nil, err
	}

  if cmd.Name != "HELLO" {
    return nil, nil, fmt.Errorf("Invalid handshake: expected hello, got %s", cmd.Name)
  }

  if len(cmd.Body) != 194 {
    return nil, nil, fmt.Errorf("Invalid hello: expected length to be 194 bytes, got %d")
  }

  if cmd.Body[0] != 1 || cmd.Body[1] != 0 {
    return nil, nil, fmt.Errorf("Expected CURVEZMQ version 1.0, got %d.%d", cmd.Body[0], cmd.Body[1])
  }

  var clientTransPubKey [32]byte
  copy(clientTransPubKey[:], cmd.Body[74:106])
  var nonce [24]byte
  copy(nonce[:], []byte("CurveZMQHELLO---"))
  copy(nonce[16:], cmd.Body[106:114])
  out := make([]byte, 0, 64)
  out, ok := box.Open(out, cmd.Body[114:], &nonce, &clientTransPubKey, &c.privKey)
  if !ok {
    return nil, nil, fmt.Errorf("Invalid signature in hello command")
  }

  if len(out) != 64 {
    return nil, nil, fmt.Errorf("Expected signature to have 64 bytes, got %d", len(out))
  }

  for idx, byte := range out {
    if byte != 0 {
      return nil, nil, fmt.Errorf("Expected signature to contain only 0's, byte %d has value %x", idx, byte)
    }
  }
  panic("hi")
  return nil, nil, nil

  //hello, err := c.awaitHello(conn)
  //if err != nil {
  //  return nil, nil, err
  //}

  //if err := c.sendWelcome(conn, hello); err != nil {
  //  return nil, nil, err
  //}

  //init, err := c.awaitInitiate(conn)
  //if err != nil {
  //  return nil, nil, err
  //}

  //if err := c.sendReady(conn, init); err != nil {
  //  return err
  //}

  //// todo finish me!
  //ret := CurveSocket{}
}

func (c *CurveClient) Handshake(conn net.Conn, meta zmtp.Metadata) (
  zmtp.Socket,
  zmtp.Metadata,
  error,
) {
  //hello := zmtp.Command{Name: "HELLO", Data: }
  return nil, nil, nil
}

func (c *Curve) SetOption(option string, val any) error {
  switch option {
  case zmtp.OptionServer:
    serv, ok := val.(bool)
    if !ok {
      return fmt.Errorf("Value for option %s must be bool, got %T", option, val)
    }

    if serv {
      c.SetupServer()
    } else {
      c.SetupClient()
    }
  case zmtp.OptionPubKey:
    var byteData []byte
    byteData, ok := val.([]byte)
    if !ok {
      strData, ok := val.(string)
      if !ok {
        return fmt.Errorf("Value for option %s must be string or []byte, got %T", option, val)
      }
      byteData = []byte(strData)
    }

    if len(byteData) != 32 {
      return fmt.Errorf("Key must be 32 bytes, got %d", len(byteData))
    }

    if c.serv != nil {
      copy(c.serv.pubKey[:], byteData)
    } else {
      if c.cli == nil {
        c.cli = &CurveClient{}
      }
      copy(c.cli.pubKey[:], byteData)
    }
  case zmtp.OptionSecKey:
    var byteData []byte
    byteData, ok := val.([]byte)
    if !ok {
      strData, ok := val.(string)
      if !ok {
        return fmt.Errorf("Value for option %s must be string or []byte, got %T", option, val)
      }
      byteData = []byte(strData)
    }

    if len(byteData) != 32 {
      return fmt.Errorf("Key must be 32 bytes, got %d", len(byteData))
    }

    if c.serv != nil {
      copy(c.serv.privKey[:], byteData)
    } else {
      if c.cli == nil {
        c.cli = &CurveClient{}
      }
      copy(c.cli.privKey[:], byteData)
    }
  case zmtp.OptionSrvKey:
    var byteData []byte
    byteData, ok := val.([]byte)
    if !ok {
      strData, ok := val.(string)
      if !ok {
        return fmt.Errorf("Value for option %s must be string or []byte, got %T", option, val)
      }
      byteData = []byte(strData)
    }

    if len(byteData) != 32 {
      return fmt.Errorf("Key must be 32 bytes, got %d", len(byteData))
    }

    if c.serv != nil {
      return fmt.Errorf("Cannot set server key on curve server (set OptionSecKey to set the private key for the server)")
    }

    if c.cli == nil {
      c.cli = &CurveClient{}
    }
    copy(c.cli.serverPubKey[:], byteData)
  }

  return nil
}

func (c *Curve) SetupServer() {
  c.serv = &CurveServer{}
  c.cli = nil
}

func (c *Curve) SetupClient() {
  c.serv = nil
  c.cli = &CurveClient{}
}

type CurveSocket struct {
  sharedKey [32]byte
	net.Conn
}

// Net returns the underlying net.Conn for the socket.
func (n CurveSocket) Net() net.Conn {
	return n.Conn
}
