package dep1

import "fmt"

type Foo struct{}

func NewFoo() *Foo {
	return &Foo{}
}

func (f *Foo) Wibble() error {
	return fmt.Errorf("This is not mocked!")
}
