package gomq

import (
	"log"
)

type PrintBus struct{}

func (PrintBus) Post(ev Event) {
	log.Printf("%s: %s <-> %s (%s)", ev.EventType, ev.LocalAddr, ev.RemoteAddr, ev.Notes)
}
