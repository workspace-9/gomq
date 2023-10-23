package main

import (
  "context"
  "log"
  "time"

  "github.com/pebbe/zmq4"
  "github.com/exe-or-death/gomq"
  _ "github.com/exe-or-death/gomq/transport/tcp"
  _ "github.com/exe-or-death/gomq/zmtp/null"
  _ "github.com/exe-or-death/gomq/types/pull"
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

  if err := sock.Connect("tcp://127.0.0.1:52849"); err != nil {
    log.Fatalf("Failed connecting to local endpoint: %s", err.Error())
  }

  for {
    msg, err := sock.Recv()
    if err != nil {
      log.Printf("Failed receiving: %s", err.Error())
    }

    for _, part := range msg {
      log.Println(string(part))
    }
  }
}

func runPushSock() {
  time.Sleep(time.Second*5)
  sock, err := zmq4.NewSocket(zmq4.PUSH)
  if err != nil {
    log.Fatalf("Failed creating push socket: %s", err.Error())
  }

  sock.Bind("tcp://127.0.0.01:52849")
  for {
    if _, err := sock.SendMessage("hello!"); err != nil {
      log.Fatalf("Failed sending message: %s", err.Error())
    }
    time.Sleep(time.Second)
  }
}
