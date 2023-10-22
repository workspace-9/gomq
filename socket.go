package gomq

import (
)

type Socket struct {
  driver driver
}

func (s Socket) SendMessage(data [][]byte) error {
  return s.driver.sendMessage(data)
}

func (s Socket) RecvMessage() ([][]byte, error) {
  return s.driver.recvMessage()
}
