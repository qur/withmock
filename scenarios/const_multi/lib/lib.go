package lib

import (
	"fmt"
)

const a, b, c int = 1, 2, 3

func Wibble() error {
	return fmt.Errorf("Not Mocked!")
}
