package code

import (
	"github.com/qur/withmock/scenarios/has_init/lib"
	"github.com/qur/withmock/scenarios/has_init/lib2"
)

func TryMe() error {
	return lib.Wibble()
}

func TryMe2() error {
	return lib2.Wibble()
}
