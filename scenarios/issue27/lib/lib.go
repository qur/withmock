package lib

import (
	"fmt"
)

type Bar interface {
	Frob() error
}

type Foo interface {
	fmt.Stringer
	Wibble() error
	Bar
}
