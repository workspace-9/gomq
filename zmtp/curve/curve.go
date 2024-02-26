package curve

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/workspace-9/gomq/zmtp"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
	"golang.org/x/crypto/nacl/secretbox"
)

type Curve struct {
	serv *CurveServer
	cli  *CurveClient
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

// todo: check client nonce fmt
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
		return nil, nil, fmt.Errorf("Invalid hello: expected length to be 194 bytes, got %d", len(cmd.Body))
	}

	if cmd.Body[0] != 1 || cmd.Body[1] != 0 {
		return nil, nil, fmt.Errorf("Expected CURVEZMQ version 1.0, got %d.%d", cmd.Body[0], cmd.Body[1])
	}

	var clientTransPubKey [32]byte
	copy(clientTransPubKey[:], cmd.Body[74:106])
	var nonce [24]byte
	copy(nonce[:], []byte("CurveZMQHELLO---"))
	copy(nonce[16:], cmd.Body[106:114])
	cliNonceIdx := binary.BigEndian.Uint64(nonce[16:])
	if cliNonceIdx != 1 {
		return nil, nil, fmt.Errorf("Expected client nonce to be 1, got %d", cliNonceIdx)
	}
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

	preCookiePubTrans, preCookieSecTrans, err := box.GenerateKey(rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("Failed generating keys: %s", err))
	}

	// at this point, we welcome
	welcome := zmtp.Command{Name: "WELCOME"}
	welcomeBody := make([]byte, 160)
	welcome.Body = welcomeBody

	var cookie [64]byte
	copy(cookie[:], clientTransPubKey[:])
	copy(cookie[32:], preCookieSecTrans[:])
	_, cookieKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("Failed creaing cookie key: %s", err.Error()))
	}
	defer func() {
		for idx := range cookieKey {
			cookieKey[idx] = 0
		}
	}()
	copy(nonce[:], []byte("COOKIE--"))
	if n, err := rand.Read(nonce[8:]); err != nil || n != 16 {
		panic(fmt.Sprintf("Failed creating cookie nonce: %s", err.Error()))
	}
	cookieData := make([]byte, 96)
	secretbox.Seal(cookieData[16:16], cookie[:], &nonce, cookieKey)
	copy(cookieData[:16], nonce[8:])

	welcomeBox := make([]byte, 128)
	copy(welcomeBox, preCookiePubTrans[:])
	copy(welcomeBox[32:], cookieData)
	copy(nonce[:], "WELCOME-")
	if n, err := rand.Read(nonce[8:]); err != nil || n != 16 {
		panic(fmt.Sprintf("Failed creating welcome box nonce: %s", err.Error()))
	}
	copy(welcomeBody, nonce[8:])
	box.Seal(welcomeBody[16:16], welcomeBox, &nonce, &clientTransPubKey, &c.privKey)
	if _, err := welcome.WriteTo(conn); err != nil {
		return nil, nil, fmt.Errorf("Failed writing welcome message: %s", err.Error())
	}

	conn.SetDeadline(time.Now().Add(time.Second * 60))
	var init zmtp.Command
	if _, err := init.ReadFrom(conn); err != nil {
		return nil, nil, fmt.Errorf("Failed reading initiate command: %s", err.Error())
	}
	conn.SetDeadline(time.Time{})

	if init.Name != "INITIATE" {
		return nil, nil, fmt.Errorf("Expected initiate command, got %s", init.Name)
	}

	if len(init.Body) < 248 {
		return nil, nil, fmt.Errorf("Expected initiate command to be at least 248 bytes long, was %d", len(init.Body))
	}

	copy(nonce[:], []byte("COOKIE--"))
	copy(nonce[8:], init.Body[:16])
	clientCookieBox := init.Body[16:96]
	clientCookieData := make([]byte, 0, 64)
	clientCookieData, ok = secretbox.Open(clientCookieData, clientCookieBox, &nonce, cookieKey)
	if !ok {
		return nil, nil, fmt.Errorf("Client sent invalid cookie")
	}

	var serverTransPubKey, serverTransSecKey [32]byte
	copy(serverTransSecKey[:], clientCookieData[32:])
	curve25519.ScalarBaseMult(&serverTransPubKey, &serverTransSecKey)

	// second point to check client short nonce
	copy(nonce[:], []byte("CurveZMQINITIATE"))
	copy(nonce[16:], init.Body[96:104])
	cliNonceIdx = binary.BigEndian.Uint64(nonce[16:])
	if cliNonceIdx != 2 {
		return nil, nil, fmt.Errorf("Expected client nonce to be 2, got %d", cliNonceIdx)
	}
	initBox := make([]byte, 0, len(init.Body)-120)
	initBox, ok = box.Open(initBox, init.Body[104:], &nonce, &clientTransPubKey, &serverTransSecKey)
	if !ok {
		return nil, nil, fmt.Errorf("Failed opening initiate box")
	}

	var clientPermPublicKey [32]byte
	copy(clientPermPublicKey[:], initBox[:32])
	vouch := initBox[32:128]
	metaData := zmtp.Metadata(initBox[128:])
	copy(nonce[:8], []byte("VOUCH---"))
	copy(nonce[8:], vouch[:16])
	vouchData := make([]byte, 0, 64)
	vouchData, ok = box.Open(vouchData, vouch[16:], &nonce, &clientPermPublicKey, &serverTransSecKey)
	if !ok {
		return nil, nil, fmt.Errorf("Failed opening vouch box")
	}

	readyCmd := zmtp.Command{Name: "READY"}
	copy(nonce[:16], "CurveZMQREADY---")
	binary.BigEndian.AppendUint64(nonce[16:16], 1)
	readyBody := make([]byte, len(meta)+16+8)
	fmt.Println(len(meta))
	meta.Properties(func(name, value string) { fmt.Println(name, value) })
	metaData.Properties(func(name, value string) { fmt.Println(name, value) })
	box.Seal(readyBody[8:8], meta, &nonce, &clientTransPubKey, &serverTransSecKey)
	binary.BigEndian.AppendUint64(readyBody[0:0], 1)
	fmt.Println(readyBody)
	readyCmd.Body = readyBody
	if _, err := readyCmd.WriteTo(conn); err != nil {
		return nil, nil, fmt.Errorf("Failed writing ready command: %s", err.Error())
	}

	ret := &CurveSocket{nonceIdx: 2, peerNonceIdx: 2, isServ: true, Conn: conn}
	box.Precompute(&ret.sharedKey, &clientTransPubKey, &serverTransSecKey)
	return ret, metaData, nil
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
	sharedKey    [32]byte
	nonceIdx     uint64
	peerNonceIdx uint64
	isServ       bool
	net.Conn
}

func (c *CurveSocket) Read() (zmtp.CommandOrMessage, error) {
	var ret zmtp.CommandOrMessage
	if _, err := ret.ReadFrom(c.Conn); err != nil {
		return ret, err
	}

	if ret.IsMessage {
		return zmtp.CommandOrMessage{}, fmt.Errorf("Cannot send raw message on curve socket")
	}

	if ret.Command.Name != "MESSAGE" {
		return ret, nil
	}

	if len(ret.Command.Body) < 25 {
		return zmtp.CommandOrMessage{}, fmt.Errorf("Received invalid message, length must be at least 25, got %d", len(ret.Command.Body))
	}

	var nonce [24]byte
	if c.isServ {
		copy(nonce[:], []byte("CurveZMQMESSAGEC"))
	} else {
		copy(nonce[:], []byte("CurveZMQMESSAGES"))
	}
	shortNonce := binary.BigEndian.Uint64(ret.Command.Body[:8])
	if shortNonce != c.peerNonceIdx+1 {
		return zmtp.CommandOrMessage{}, fmt.Errorf("Peer used invalid nonce (expected %d, got %d)", c.peerNonceIdx+1, shortNonce)
	}
	c.peerNonceIdx++
	copy(nonce[16:], ret.Command.Body[:8])
	out := make([]byte, len(ret.Command.Body)-24)
	out, ok := box.OpenAfterPrecomputation(out[0:0], ret.Command.Body[8:], &nonce, &c.sharedKey)
	if !ok {
		return zmtp.CommandOrMessage{}, fmt.Errorf("Failed opening message box")
	}

	return zmtp.CommandOrMessage{
		IsMessage: true, Message: &zmtp.Message{
			More: (out[0] & 0x1) == 0x1, Body: out[1:],
		},
	}, nil
}

func (c *CurveSocket) SendCommand(cmd zmtp.Command) error {
	if cmd.Name != "ERROR" {
		return fmt.Errorf("Expected error command, got %s", cmd.Name)
	}

	_, err := cmd.WriteTo(c.Conn)
	return err
}

func (c *CurveSocket) SendMessage(msg zmtp.Message) error {
	defer func() { c.nonceIdx++ }()
	cmd := zmtp.Command{Name: "MESSAGE"}
	body := make([]byte, 8+17+len(msg.Body))
	binary.BigEndian.AppendUint64(body[0:0], c.nonceIdx)

	var nonce [24]byte
	if c.isServ {
		copy(nonce[:], []byte("CurveZMQMESSAGES"))
	} else {
		copy(nonce[:], []byte("CurveZMQMESSAGEC"))
	}
	binary.BigEndian.AppendUint64(nonce[16:16], c.nonceIdx)
	toSeal := make([]byte, 1+len(msg.Body))
	if msg.More {
		toSeal[0] = 0x1
	}
	copy(toSeal[1:], msg.Body)
	box.SealAfterPrecomputation(body[8:8], toSeal, &nonce, &c.sharedKey)
	cmd.Body = body
	fmt.Println(len(cmd.Body), cmd.Body)
	_, err := cmd.WriteTo(c.Conn)
	return err
}

func (c *CurveSocket) Close() error {
	return c.Conn.Close()
}

// Net returns the underlying net.Conn for the socket.
func (n *CurveSocket) Net() net.Conn {
	return n.Conn
}
