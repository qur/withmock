package lib

import (
	"fmt"
)

var foo = fmt.Errorf("init not called")

func init() {
	foo = nil
}

func init() {
	fmt.Println("second init does nothing important")
}

func Wibble() error {
	return foo
}
