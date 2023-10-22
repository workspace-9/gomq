package main

import (
	"context"
	"log"
	"time"

	"github.com/exe-or-death/gomq/transport/tcp"
	"github.com/exe-or-death/gomq/zmtp"
	"github.com/exe-or-death/gomq/zmtp/null"
	"github.com/pebbe/zmq4"
)

func testPull() {
	pebSock, _ := zmq4.NewSocket(zmq4.PUSH)
	pebSock.Bind("tcp://127.0.0.1:52849")

	go func() {
		for {
			_, err := pebSock.SendMessage("hello!")
			if err != nil {
				log.Println(err)
			}
      log.Println("Sent")
      time.Sleep(time.Second)
		}
	}()

	conn, err := tcp.TCPTransport{}.Connect(context.Background(), "127.0.0.1:52849")
	if err != nil {
		log.Fatalf("Failed connecting to remote socket: %s", err.Error())
	}

	mech := null.Null{}
	greeting := zmtp.NewGreeting()
	greeting.SetMechanism(mech.Name())
	greeting.SetVersionMajor(3)
	greeting.SetVersionMinor(1)
	if _, err := greeting.WriteTo(conn); err != nil {
		log.Fatalf("Failed writing greeting: %s", err.Error())
	}

	var respGreeting zmtp.Greeting
	if _, err := respGreeting.ReadFrom(conn); err != nil {
		log.Fatalf("Failed reading greeting: %s", err.Error())
	}

	if err := mech.ValidateGreeting(&respGreeting); err != nil {
		log.Fatalf("Failed validating greeting: %s", err.Error())
	}

	props := zmtp.Metadata{}
	props.AddProperty("Socket-Type", "PULL")
	socket, respProps, err := mech.Handshake(conn, props)
	if err != nil {
		log.Fatalf("Failed handshake: %s", err.Error())
	}

	respPropValues, err := respProps.Properties()
	if err != nil {
		log.Fatalf("Failed reading response properties: %s", err.Error())
	}
	for _, v := range respPropValues {
		log.Printf("Detected prop: %s=%s", v[0], v[1])
	}

  data, err := socket.Read()
  if err != nil {
    log.Fatalf("Failed reading message: %s", err.Error())
  }

  if !data.IsMessage {
    log.Fatalf("Expected message, received command")
  }

  log.Println(string(data.Message.Body))
}
