package code

import (
	"testing"

	"code.google.com/p/gomock/gomock"

	"github.com/qur/withmock/scenarios/issue19/lib1" // mock
	"github.com/qur/withmock/scenarios/issue19/lib2" // mock
)

func TestTryMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib1.MOCK_DEFAULT().SetController(ctrl)
	lib1.EXPECT().Wibble1().Return(nil)

	// Run the function we want to test
	err := TryMe1()

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}

func TestTryMe2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib2.MOCK2().SetController(ctrl)
	lib2.EXPECT2().Wibble2().Return(nil)

	// Run the function we want to test
	err := TryMe2()

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}

func TestTryMe3(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib2.MOCK2().SetController(ctrl)
	f := &lib2.Foo{}
	f.EXPECT3().Wibble().Return(nil)

	// Run the function we want to test
	err := TryMe3(f)

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}
