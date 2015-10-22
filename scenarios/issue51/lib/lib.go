package lib

import (
	"fmt"
)

type Test struct{}

func (t Test) Method() error {
	return fmt.Errorf("Raw Method")
}

func (t *Test) PointerMethod() error {
	return fmt.Errorf("Raw PointerMethod")
}
