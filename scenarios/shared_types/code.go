package code

import (
	"github.com/qur/withmock/scenarios/shared_types/dep"
)

func TryMe(a, b int) error {
	return dep.Wibble(a, b)
}

func TryMe2(x, y int) (a, b int) {
	return dep.Bar(x, y)
}
