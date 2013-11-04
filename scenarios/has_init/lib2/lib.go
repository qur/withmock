package lib2

import (
	"fmt"
)

var foo = fmt.Errorf("init not called")

func init() {
	foo = nil
}

func Wibble() error {
	return foo
}
