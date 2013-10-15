package code

import (
	"github.com/qur/withmock/scenarios/issue17/lib_asm1"
	"github.com/qur/withmock/scenarios/issue17/lib_asm2"
	"github.com/qur/withmock/scenarios/issue17/lib_c1"
	"github.com/qur/withmock/scenarios/issue17/lib_c2"
)

func TryMeAsm1() bool {
	return lib_asm1.Wibble()
}

func TryMeAsm2() bool {
	return lib_asm2.Wibble()
}

func TryMeC1() bool {
	return lib_c1.Wibble()
}

func TryMeC2() bool {
	return lib_c2.Wibble()
}
