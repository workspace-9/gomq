package curve

import (
	"fmt"
	"net"

	"github.com/workspace-9/gomq/zmtp"
)

type Curve struct {
	serv *CurveServer
	cli  *CurveClient
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
