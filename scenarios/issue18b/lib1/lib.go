package lib1

import "github.com/qur/withmock/scenarios/issue18/lib2"

func Wibble() bool

func Bar() error {
	return lib2.Bar()
}
