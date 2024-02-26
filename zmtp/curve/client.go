package curve

import (
	"fmt"
	"net"

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
	var transPub, transPriv [32]byte
	var nonce Nonce
	GenerateKeys(&transPub, &transPriv)

	hello := zmtp.Command{
		Name: "HELLO",
	}
	body := make([]byte, 194)
	hello.Body = body
	body[0] = 1
	copy(body[74:106], transPub[:])
	nonce.Short("CurveZMQHELLO---", 1)
	var sigBox [64]byte
	box.Seal(body[114:114], sigBox[:], nonce.N(), &c.serverPubKey, &transPriv)
	if _, err := hello.WriteTo(conn); err != nil {
		return nil, nil, fmt.Errorf("Failed sending hello to server: %w", err)
	}

	return nil, nil, nil
}
