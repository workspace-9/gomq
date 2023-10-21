package main

import (
  "fmt"
  "net"
  "time"
  "github.com/pebbe/zmq4"
  "github.com/exe-or-death/gomq/null"
  "github.com/exe-or-death/gomq/zmtp"
)

func main() {
  pebSock, err := zmq4.NewSocket(zmq4.PULL)
  try(err)
  defer pebSock.Close()

  monSock, err := zmq4.NewSocket(zmq4.PAIR)
  try(err)
  monSock.Connect("inproc://monitor.sock")
  go func() {
    for {
      a, b, c, err := monSock.RecvEvent(0)
      try(err)
      fmt.Println(a, b, c)
    }
  }()

  try(pebSock.Bind("tcp://127.0.0.1:5284"))
  try(pebSock.Monitor("inproc://monitor.sock", zmq4.EVENT_ALL))
  go func() {
    for {
      fmt.Println("running...")
      msg, err := pebSock.RecvMessage(0)
      try(err)
      fmt.Println("here", string(msg[0]))
    }
  }()

  conn, err := net.Dial("tcp", "127.0.0.1:5284")
  try(err)
  defer conn.Close()

  try(null.SetupNull(conn, map[string]string{"Socket-Type": "PUSH"}))

  message := zmtp.Message("hey!")
  try(message.Send(conn, false))
  try(message.Send(conn, false))
  try(message.Send(conn, false))
  try(message.Send(conn, false))
  try(message.Send(conn, false))
  time.Sleep(time.Second)
}

func try(err error) {
  if err != nil {
    panic(err)
  }
}
