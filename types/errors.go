package types

type operationNotPermitted struct{}

func (operationNotPermitted) Error() string {
  return "Operation not permitted"
}

var ErrOperationNotPermitted operationNotPermitted

type allPeersBusy struct {}

func (allPeersBusy) Error() string {
  return "All peers busy"
}

var ErrAllPeersBusy allPeersBusy

type alreadyConnected struct {}

func (alreadyConnected) Error() string {
  return "Already connected"
}

var ErrAlreadyConnected alreadyConnected

type alreadyBound struct {}

func (alreadyBound) Error() string {
  return "Already bound"
}

var ErrAlreadyBound alreadyBound

type neverConnected struct {}

func (neverConnected) Error() string {
  return "Never connected"
}

var ErrNeverConnected neverConnected

type neverBound struct {}

func (neverBound) Error() string {
  return "Never bound"
}

var ErrNeverBound neverBound
