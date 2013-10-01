package code

import (
	"github.com/qur/withmock/scenarios/new_methods/lib"
)

func TryMe(foo lib.Foo) error {
	return foo.Wibble()
}
