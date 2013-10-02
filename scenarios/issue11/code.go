package code

import (
	"github.com/qur/withmock/scenarios/issue11/dep1"
	"github.com/qur/withmock/scenarios/issue11/dep2"
)

func TryMe() error {
	foo := dep1.NewFoo()
	return dep2.Wibble(foo)
}
