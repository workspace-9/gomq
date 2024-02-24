package null

import (
	"github.com/workspace-9/gomq"
	"github.com/workspace-9/gomq/zmtp"
)

const MechName = "NULL"

func init() {
	gomq.RegisterMechanism(MechName, func() zmtp.Mechanism {
		return Null{}
	})
}
