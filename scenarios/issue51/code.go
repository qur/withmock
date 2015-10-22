package code

import (
	"github.com/qur/withmock/scenarios/issue51/lib"
)

func TryMe(t *lib.Test) error {
	return t.PointerMethod()
}
