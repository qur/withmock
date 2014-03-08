package code

import (
	"testing"

	"github.com/qur/gomock/gomock"

	"github.com/qur/withmock/scenarios/issue17/lib_asm2" // mock
	"github.com/qur/withmock/scenarios/issue17/lib_c2" // mock
)

func TestTryMeAsm1(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Run the function we want to test
	ret := TryMeAsm1()

	if ret {
		t.Error("Asm 1: Expected false, got true")
	}
}

func TestTryMeAsm2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib_asm2.MOCK().SetController(ctrl)
	lib_asm2.EXPECT().Wibble().Return(true)

	// Run the function we want to test
	ret := TryMeAsm2()

	if !ret {
		t.Error("Asm 2: Expected true, got false")
	}
}

func TestTryMeC1(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Run the function we want to test
	ret := TryMeC1()

	if ret {
		t.Error("C 1: Expected false, got true")
	}
}

func TestTryMeC2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib_c2.MOCK().SetController(ctrl)
	lib_c2.EXPECT().Wibble().Return(true)

	// Run the function we want to test
	ret := TryMeC2()

	if !ret {
		t.Error("C 2: Expected true, got false")
	}
}
