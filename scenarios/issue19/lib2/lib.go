package lib2

import (
	"fmt"
)

func Wibble2() error {
	return fmt.Errorf("Not Mocked!")
}

type Foo struct{}

func (f *Foo) Wibble() error {
	return fmt.Errorf("Not Mocked!")
}
