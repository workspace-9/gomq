package zmtp

import (
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
  if _, err := w.Write([]byte{
    0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x7F,
    g.Version.Major, g.Version.Minor,
  }); err != nil {
    return err
  }

  mechanismBytes, err := io.WriteString(w, g.Mechanism)
  if err != nil {
    return err
  }
  empty := [31]byte{}
  if _, err := w.Write(empty[:20 - mechanismBytes]); err != nil {
    return err
  }

  if g.Server {
    if _, err := w.Write([]byte{1}); err != nil {
      return err
    }
  } else if _, err := w.Write([]byte{0}); err != nil {
    return err
  }

  if _, err := w.Write(empty[:]); err != nil {
    return err
  }

  return nil
}
