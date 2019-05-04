package control

type ExitError interface {
	ExitCode() int
}

type wrapper struct {
	err  error
	code int
}

func (w wrapper) Error() string {
	return w.err.Error()
}

func (w wrapper) Cause() error {
	return w.err
}

func (w wrapper) ExitCode() int {
	return w.code
}

func Exited(err error, code int) error {
	return wrapper{
		err:  err,
		code: code,
	}
}
