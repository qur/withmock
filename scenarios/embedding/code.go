package code

import (
	"github.com/qur/withmock/scenarios/embedding/lib"
)

func TryMe() error {
	foo := lib.NewFoo()
	return foo.Wibble()
}
