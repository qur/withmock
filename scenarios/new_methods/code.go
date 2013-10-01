package code

import (
	"github.com/qur/withmock/scenarios/new_methods/lib"
)

func TryMe() error {
	foo := lib.NewFoo()
	bar := lib.NewBar()
	bar.Wibble()
	return foo.Wibble()
}
