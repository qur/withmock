package lib

import (
	"fmt"
)

type Foo struct {
	Bar
}

type Bar struct{}

func NewFoo() *Foo {
	return &Foo{}
}

func (b *Bar) Wibble() error {
	return fmt.Errorf("Not Mocked!")
}
