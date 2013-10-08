package code

import (
	"github.com/qur/withmock/scenarios/func_literals/lib"
)

func TryMe() error {
	lib.Wibble()
	return lib.Bar()
}
