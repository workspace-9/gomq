package main

import (
  "context"
  "log"
  "time"

  "github.com/exe-or-death/gomq"
  _ "github.com/exe-or-death/gomq/transport/tcp"
  _ "github.com/exe-or-death/gomq/zmtp/null"
  _ "github.com/exe-or-death/gomq/types/pull"
  _ "github.com/exe-or-death/gomq/types/push"
)

func main() {
  go runPushSock()
  runPullSock()
}

func runPullSock() {
  ctx := gomq.NewContext(context.Background())
  sock, err := ctx.NewSocket("PULL", "NULL")
  if err != nil {
    log.Fatalf("Failed creating socket: %s", err.Error())
  }

  if err := sock.Bind("tcp://127.0.0.1:52849"); err != nil {
    log.Fatalf("Failed connecting to local endpoint: %s", err.Error())
  }

  for idx := 0; idx < 2; idx++ {
    msg, err := sock.Recv()
    if err != nil {
      log.Printf("Failed receiving: %s", err.Error())
    }

    for _, part := range msg {
      log.Println("RX:", string(part))
    }
  }
}

func runPushSock() {
  time.Sleep(time.Second)
  ctx := gomq.NewContext(context.Background())
  sock, err := ctx.NewSocket("PUSH", "NULL")
  if err != nil {
    log.Fatalf("Failed creating socket: %s", err.Error())
  }

  if err := sock.Connect("tcp://127.0.0.1:52849"); err != nil {
    log.Fatalf("Failed binding to local endpoint: %s", err.Error())
  }

  if err := sock.Send([][]byte{[]byte("hola!")}); err != nil {
    log.Fatalf("Failed sending: %s", err.Error())
  }

  if err := sock.Disconnect("tcp://127.0.0.1:52849"); err != nil {
    log.Fatalf("Failed disconnecting: %s", err.Error())
  }

  time.Sleep(time.Second)
  if err := sock.Connect("tcp://127.0.0.1:52849"); err != nil {
    log.Fatalf("Failed connecting to local endpoint: %s", err.Error())
  }

  if err := sock.Send([][]byte{[]byte("hola!")}); err != nil {
    log.Fatalf("Failed sending: %s", err.Error())
  }
  sock.Close()
}
