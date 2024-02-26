package curve

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/workspace-9/gomq/zmtp"
	"golang.org/x/crypto/nacl/box"
)

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

	//fixme
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
	_, err := cmd.WriteTo(&overrideFirstByteWriter{
		f: func(old byte) byte {
			return old - 4
		},
		writtenFirstByte: false,
		Writer:           c.Conn,
	})
	return err
}

type overrideFirstByteWriter struct {
	f                func(old byte) byte
	writtenFirstByte bool
	io.Writer
}

func (w *overrideFirstByteWriter) Write(p []byte) (n int, err error) {
	if w.writtenFirstByte {
		return w.Writer.Write(p)
	}

	if len(p) == 0 {
		return 0, nil
	}
	var firstByte [1]byte
	firstByte[0] = w.f(p[0])
	_, err = w.Writer.Write(firstByte[:])
	if err != nil {
		return 0, err
	}
	w.writtenFirstByte = true

	n, err = w.Writer.Write(p[1:])
	n++
	return
}

func (c *CurveSocket) Close() error {
	return c.Conn.Close()
}

// Net returns the underlying net.Conn for the socket.
func (n *CurveSocket) Net() net.Conn {
	return n.Conn
}
