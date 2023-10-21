package zmtp

import (
  "fmt"
  "io"
)

type Greeting struct {
  // Version represents the version of zmtp protocol.
  Version struct {
    // Major protocol version.
    Major uint8

    // Minor protocol version.
    Minor uint8
  }

  // Mechanism of security for the following communication.
  Mechanism string

  // Server is true if the socket is bound.
  Server bool
}

// Send a greeting over a writer.
func (g *Greeting) Send(w io.Writer) error {
  total := 0
  if n, err := w.Write([]byte{
    0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x7F,
    g.Version.Major, g.Version.Minor,
  }); err != nil {
    return err
  } else {
    total += n
  }

  mechanismBytes, err := io.WriteString(w, g.Mechanism)
  if err != nil {
    return err
  }
  total += mechanismBytes
  empty := [31]byte{}
  if _, err := w.Write(empty[:20 - mechanismBytes]); err != nil {
    return err
  }
  total += len(empty[:20-mechanismBytes])

  if g.Server {
    if _, err := w.Write([]byte{1}); err != nil {
      return err
    }
  } else if _, err := w.Write([]byte{0}); err != nil {
    return err
  }
  total += 1

  if n, err := w.Write(empty[:]); err != nil {
    return err
  } else {
    total += n
  }
  fmt.Println("wrote", total, "bytes in greeting")

  return nil
}

type invalidGreeting struct{}

func (invalidGreeting) Error() string {
  return "Invalid greeting"
}

var ErrInvalidGreeting invalidGreeting

// ReadGreeting reads a greeting from a stream.
func ReadGreeting(r io.Reader) (*Greeting, error) {
  var data [64]byte
  if n, err := io.ReadFull(r, data[:]); n != 64 {
    return nil, ErrInvalidGreeting
  } else if err != nil {
    return nil, err
  }

  greeting := &Greeting{}
  greeting.Version.Major = data[10]
  greeting.Version.Minor = data[11]
  mechanismBytes := data[12:32]
  for idx, b := range mechanismBytes {
    if b == 0 {
      greeting.Mechanism = string(mechanismBytes[:idx])
      break
    }
  }
  greeting.Server = data[33] == 1
  return greeting, nil
}
