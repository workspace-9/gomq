package null

import (
	"github.com/exe-or-death/gomq"
	"github.com/exe-or-death/gomq/zmtp"
)

func init() {
	gomq.RegisterMechanism("NULL", func() zmtp.Mechanism {
		return Null{}
	})
}
