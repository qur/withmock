package code

import (
	"github.com/qur/withmock/scenarios/interface/lib"
)

func TryMe(foo lib.Foo) error {
	return foo.Wibble()
}
