package lib

import (
	"fmt"
)

var Wibble = func() {
	fmt.Printf("lib.Wibble")
}

func Bar() error {
	return fmt.Errorf("Not Mocked!")
}
