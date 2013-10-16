package code

import (
	. "github.com/qur/withmock/scenarios/issue19/lib1"
	. "github.com/qur/withmock/scenarios/issue19/lib2"
)

func TryMe1() error {
	return Wibble1()
}

func TryMe2() error {
	return Wibble2()
}
