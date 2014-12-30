package code

import (
	"github.com/qur/withmock/scenarios/issue17b/lib_asm1"
	"github.com/qur/withmock/scenarios/issue17b/lib_asm2"
)

func TryMeAsm1() bool {
	return lib_asm1.Wibble()
}

func TryMeAsm2() bool {
	return lib_asm2.Wibble()
}
