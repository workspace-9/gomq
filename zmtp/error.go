package zmtp

type Error struct {
  error
  Fatal bool
}

func Fatal(err error) Error {
  return Error{
    error: err, Fatal: true,
  }
}

func Fail(err error) Error {
  return Error{
    error: err, Fatal: false,
  }
}
