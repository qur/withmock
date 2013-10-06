package code

import (
	"github.com/qur/withmock/scenarios/runtime/dep"
)

func TryMe(a, b int) int {
	return dep.Wibble(a, b)
}

func TryMe2(a, b int) int {
	return dep.Bar(a, b)
}

func TryMe3(a, b int) int {
	f := dep.NewFoo(a)
	return f.Wibble(b)
}
