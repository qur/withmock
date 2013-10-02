package dep2

import (
	"github.com/qur/withmock/scenarios/issue11/dep1"
)

func Wibble(foo *dep1.Foo) error {
	return foo.Wibble()
}
