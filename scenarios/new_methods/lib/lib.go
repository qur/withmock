package lib

import (
	"fmt"
)

type Foo interface {
	Wibble() error
}

type foo struct{}

type bar int

func NewFoo() Foo {
	return &foo{}
}

func NewBar() Foo {
	return bar(0)
}

func (f *foo) Wibble() error {
	return fmt.Errorf("Not Mocked!")
}

func (b bar) Wibble() error {
	return fmt.Errorf("Not Mocked!")
}
