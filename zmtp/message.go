package zmtp

import (
  "encoding/binary"
  "io"
)

// Message is a slice of bytes.
type Message []byte

// Send a message over a writer.
// If more is true then more messages will be expected in this frame.
func (m Message) Send(w io.Writer, more bool) error {
  if len(m) <= 255 {
    return m.sendShort(w, more)
  }

  return m.sendLong(w, more)
}

func (m Message) sendShort(w io.Writer, more bool) error {
  if more {
    if _, err := w.Write([]byte{0x01, uint8(len(m))}); err != nil {
      return err
    }
  } else if _, err := w.Write([]byte{0x00, uint8(len(m))}); err != nil {
    return err
  }

  _, err := w.Write(m)
  return err
}

func (m Message) sendLong(w io.Writer, more bool) error {
  if more {
    if _, err := w.Write([]byte{0x03}); err != nil {
      return err
    }
  } else if _, err := w.Write([]byte{0x02}); err != nil {
    return err
  }

  
  if err := binary.Write(w, binary.BigEndian, uint64(len(m))); err != nil {
    return err
  }
  _, err := w.Write(m)
  return err
}

type badMessageHeader struct{}

func (badMessageHeader) Error() string {
  return "Bad message header"
}

var ErrBadMessageHeader badMessageHeader

// ReadMessage reads a message from the reader.
func ReadMessage(r io.Reader) (m Message, more bool, err error) {
  var buffer [1]byte
  if n, err := r.Read(buffer[:1]); n != 1 || err != nil {
    return nil, false, err
  }

  var shortSize bool
  switch buffer[0] {
  case 0x00:
    shortSize = true
    more = false
  case 0x01:
    shortSize = true
    more = true
  case 0x02:
    shortSize = false
    more = false
  case 0x03:
    shortSize = false
    more = true
  default:
    err = ErrBadMessageHeader
    return
  }

  var messageLen uint64
  if shortSize {
    if n, err := r.Read(buffer[:1]); n != 1 || err != nil {
      return nil, false, err
    }
    messageLen = uint64(buffer[0])
  } else {
    if err = binary.Read(r, binary.BigEndian, &messageLen); err != nil {
      return
    }
  }

  m = make([]byte, messageLen)
  _, err = io.ReadFull(r, m)
  return
}
