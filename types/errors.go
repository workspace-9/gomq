package types

type operationNotPermitted struct{}

func (operationNotPermitted) Error() string {
  return "Operation not permitted"
}

var ErrOperationNotPermitted operationNotPermitted
