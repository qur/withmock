package lib

import (
	"fmt"
)

var a, b, c int = 1, 2, 3

func Wibble() error {
	return fmt.Errorf("Not Mocked!")
}
