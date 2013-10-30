package code

import (
	"os"

	"github.com/qur/withmock/scenarios/build_constraints/lib"
)

func TryMe() error {
	return lib.Wibble()
}

func TryMe2(f *os.File) {
	f.WriteString("Hello")
}
