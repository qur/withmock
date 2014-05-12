package lib

import (
	"fmt"
)

var one = []string{
	"one",
	"two",
	"three",
	"four",
}

var two = one[1:2]

func Wibble() error {
	return fmt.Errorf("Not Mocked!")
}
