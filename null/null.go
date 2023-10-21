package null

import (
  "bytes"
  "encoding/binary"
  "fmt"
  "io"
  "github.com/exe-or-death/gomq/zmtp"
)

func SetupNull(conn io.ReadWriter, properties map[string]string) error {
  greeting := zmtp.Greeting{}
  greeting.Version.Major = 3
  greeting.Version.Minor = 1
  greeting.Mechanism = "NULL"
  greeting.Server = false

  if err := greeting.Send(conn); err != nil {
    return err
  }

  respGreeting, err := zmtp.ReadGreeting(conn)
  if err != nil {
    return err
  }

  fmt.Println("rx greeting", respGreeting)

  cmd := zmtp.Command{}
  cmd.Name = "READY"
  cmd.Body = make([]byte, 0)
  for k, v := range properties {
    nameLen := len(k)
    cmd.Body = append(cmd.Body, uint8(nameLen))
    cmd.Body = append(cmd.Body, []byte(k)...)
    cmd.Body = append(cmd.Body, 0, 0, 0, 0)
    binary.BigEndian.PutUint32(cmd.Body[len(cmd.Body)-4:], uint32(len(v)))
    cmd.Body = append(cmd.Body, []byte(v)...)
  }

  buf := &bytes.Buffer{}
  if err := cmd.Send(buf); err != nil {
    return err
  }
  readBack, err := zmtp.ReadCommand(buf)
  if err != nil {
    return err
  }
  if readBack.Name != cmd.Name {
    panic(readBack.Name + " " + cmd.Name)
  }
  if len(readBack.Body) != len(cmd.Body) {
    panic(len(readBack.Body))
  }
  {
  idx := 0
  for idx < len(readBack.Body) {
    propertyLen := readBack.Body[idx]
    idx += 1
    propertyData := string(readBack.Body[idx:idx+int(propertyLen)])
    fmt.Println(propertyData)
    idx += int(propertyLen)
    var valueLen uint32
    valueLen = binary.BigEndian.Uint32(readBack.Body[idx:idx+4])
    idx += 4
    valueData := string(readBack.Body[idx:idx+int(valueLen)])
    idx += int(valueLen)
    fmt.Println(valueData)
  }
  }

  if err := cmd.Send(conn); err != nil {
    return err
  }

  respCmd, err := zmtp.ReadCommand(conn)
  if err != nil {
    return err
  }

  idx := 0
  for idx < len(respCmd.Body) {
    propertyLen := respCmd.Body[idx]
    idx += 1
    propertyData := string(respCmd.Body[idx:idx+int(propertyLen)])
    fmt.Println(propertyData)
    idx += int(propertyLen)
    var valueLen uint32
    valueLen = binary.BigEndian.Uint32(respCmd.Body[idx:idx+4])
    idx += 4
    valueData := string(respCmd.Body[idx:idx+int(valueLen)])
    idx += int(valueLen)
    fmt.Println(valueData)
  }

  fmt.Println("rx cmd", respCmd)

  return nil
}
