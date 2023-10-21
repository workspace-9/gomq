package zmtp

import (
  "encoding/binary"
  "fmt"
  "io"
)

type CommandSize int

const (
  CommandSizeShort = 0x04
  CommandSizeLong = 0x06
)

type Command struct {
  Name string
  Body []byte
}

// WriteTo writes the command to a writer.
func (c Command) Send(w io.Writer) error {
  nameBytes := uint64(len(c.Name)) + 1
  bodyBytes := uint64(len(c.Body))
  total := nameBytes + bodyBytes
  if total <= 255 {
    return c.writeShortCommand(w, uint8(total))
  }
  return c.writeLongCommand(w, total)
}

// writeShortCommand writes a short command to the writer.
func (c Command) writeShortCommand(w io.Writer, length uint8) error {
  fmt.Printf("write short command (length=%d)\n", length)
  if _, err := w.Write([]byte{
    CommandSizeShort, length, byte(len(c.Name)),
  }); err != nil {
    return err
  }

  if _, err := io.WriteString(w, c.Name); err != nil {
    return err
  }

  if _, err := w.Write(c.Body); err != nil {
    return err
  }

  return nil
}

// writeLongCommand writes a long command to the writer.
func (c Command) writeLongCommand(w io.Writer, length uint64) error {
  fmt.Println("write long command")
  if _, err := w.Write([]byte{CommandSizeLong}); err != nil {
    return err
  }

  if err := binary.Write(w, binary.BigEndian, length); err != nil {
    return err
  }

  if _, err := w.Write([]byte{byte(len(c.Name))}); err != nil {
    return err
  }

  if _, err := io.WriteString(w, c.Name); err != nil {
    return err
  }

  if _, err := w.Write(c.Body); err != nil {
    return err
  }

  return nil
}

type badCommandHeader struct{}

func (badCommandHeader) Error() string {
  return "Bad command header"
}

var ErrBadCommandHeader badCommandHeader

func ReadCommand(r io.Reader) (c Command, err error) {
  var buffer [1]byte
  if _, err = r.Read(buffer[:1]); err != nil {
    return
  }

  var commandSize uint64
  switch buffer[0] {
  case CommandSizeLong:
    if err = binary.Read(r, binary.BigEndian, &commandSize); err != nil {
      return
    }
  case CommandSizeShort:
    if _, err = r.Read(buffer[:1]); err != nil {
      return
    }

    commandSize = uint64(buffer[0])
  default:
    err = ErrBadMessageHeader
    return
  }

  body := make([]byte, commandSize)
  if _, err = r.Read(body); err != nil {
    return
  }

  nameLen := body[0]
  c.Name = string(body[1:nameLen+1])
  c.Body = body[nameLen+1:]
  return
}
