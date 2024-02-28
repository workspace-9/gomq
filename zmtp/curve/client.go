package curve

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/workspace-9/gomq/zmtp"
	"golang.org/x/crypto/nacl/box"
)

type CurveClient struct {
	pubKey, privKey, serverPubKey [32]byte
}

func (c *CurveClient) Handshake(conn net.Conn, meta zmtp.Metadata) (
	zmtp.Socket,
	zmtp.Metadata,
	error,
) {
	var servTransPub, transPub, transPriv [32]byte
	var nonce Nonce
	GenerateKeys(&transPub, &transPriv)

	// hello!
	hello := zmtp.Command{
		Name: "HELLO",
	}
	body := make([]byte, 194)
	hello.Body = body
	body[0] = 1
	copy(body[74:106], transPub[:])
	body[113] = 1
	nonce.Short("CurveZMQHELLO---", 1)
	var sigBox [64]byte
	box.Seal(body[114:114], sigBox[:], nonce.N(), &c.serverPubKey, &transPriv)
	if _, err := hello.WriteTo(conn); err != nil {
		return nil, nil, fmt.Errorf("Failed sending hello to server: %w", err)
	}
	time.Sleep(time.Second)

	// welcome
	welcome := zmtp.Command{}
	if _, err := welcome.ReadFrom(conn); err != nil {
		return nil, nil, fmt.Errorf("Failed receiving welcome message: %s", err.Error())
	}
	if welcome.Name != "WELCOME" {
		return nil, nil, fmt.Errorf("Expected WELCOME command, got %s", welcome.Name)
	}
	nonce.FromLong("WELCOME-", welcome.Body[:16])
	welcomeBox := make([]byte, 128)
	_, ok := box.Open(welcomeBox[0:0], welcome.Body[16:], nonce.N(), &c.serverPubKey, &transPriv)
	if !ok {
		return nil, nil, fmt.Errorf("Failed opening welcome box")
	}
	copy(servTransPub[:], welcomeBox[:32])
	servCookie := welcomeBox[32:]

	// initiate
	initiate := zmtp.Command{Name: "INITIATE"}
	initiateBody := make([]byte, 96+8+32+96+len(meta)+16)
	copy(initiateBody[:96], servCookie)
	initiateBody[103] = 2

	// initiate::vouch
	nonce.Long("VOUCH---")
	vouch := make([]byte, 64)
	copy(vouch, transPub[:])
	copy(vouch[32:], c.serverPubKey[:])
	vouchBox := make([]byte, 80)
	box.Seal(vouchBox[0:0], vouch, nonce.N(), &servTransPub, &c.privKey)

	initBox := make([]byte, 128+len(meta))
	copy(initBox, c.pubKey[:])
	copy(initBox[32:48], nonce[8:])
	copy(initBox[48:128], vouchBox)
	copy(initBox[128:], meta)
	nonce.Short("CurveZMQINITIATE", 2)
	box.Seal(initiateBody[104:104], initBox, nonce.N(), &servTransPub, &transPriv)
	initiate.Body = initiateBody
	if _, err := initiate.WriteTo(conn); err != nil {
		return nil, nil, err
	}

	// ready
	var ready zmtp.Command
	if _, err := ready.ReadFrom(conn); err != nil {
		return nil, nil, err
	}
	if ready.Name != "READY" {
		return nil, nil, fmt.Errorf("Expected READY command, got %s", ready.Name)
	}
	if len(ready.Body) < 24 {
		return nil, nil, fmt.Errorf("Expected at least 24 bytes in ready command, got %d", len(ready.Body))
	}
	servNonce := binary.BigEndian.Uint64(ready.Body[:8])
	if servNonce != 1 {
		return nil, nil, fmt.Errorf("Expected server nonce to be 1, got %d", servNonce)
	}
	nonce.Short("CurveZMQREADY---", 1)
	servMeta := make([]byte, len(ready.Body)-24)
	if _, ok := box.Open(servMeta[0:0], ready.Body[8:], nonce.N(), &servTransPub, &transPriv); !ok {
		return nil, nil, fmt.Errorf("Failed opening metadata")
	}

	ret := &CurveSocket{nonceIdx: 3, peerNonceIdx: 2, isServ: false, Conn: conn}
	box.Precompute(&ret.sharedKey, &servTransPub, &transPriv)
	return ret, servMeta, nil
}
