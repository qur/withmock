package code

import (
	"github.com/qur/withmock/scenarios/issue23/shared"
	"github.com/qur/withmock/scenarios/issue23/pkg1/lib"
)

func TryMe() error {
	return lib.Wibble()
}

func TryMe2() int {
	return shared.Wibble()
}
