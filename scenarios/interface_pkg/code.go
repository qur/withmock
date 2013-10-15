package code

import (
	"github.com/qur/withmock/scenarios/interface_pkg/lib"
)

func TryMe(foo lib.Foo) error {
	return foo.Wibble()
}

type Noisy interface {
	Tooter
	IsQuiet() bool
	NewFoo() lib.Foo
}

type Tooter interface {
	Toot() error
}

func TryMe2(foo Tooter) error {
	return foo.Toot()
}
