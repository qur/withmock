package lib

// +build something

import (
	"fmt"
)

func Wibble() error {
	return fmt.Errorf("Not Mocked!")
}
