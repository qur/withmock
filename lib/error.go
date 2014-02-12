package lib

type Cerr struct {
	Ctxt string
	Err  error
}

func (c Cerr) Error() string {
	return c.Err.Error()
}

func (c Cerr) Context() string {
	if c2, ok := c.Err.(Cerr); ok {
		return c.Ctxt + ":" + c2.Context()
	} else {
		return c.Ctxt
	}
}
