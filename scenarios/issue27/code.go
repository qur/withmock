package code

import (
	"github.com/qur/withmock/scenarios/issue27/lib"
)

func TryMe(foo lib.Foo) error {
	return foo.Wibble()
}
