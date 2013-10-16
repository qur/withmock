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

	lib1.MOCK().SetController(ctrl)
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

	lib2.MOCK().SetController(ctrl)
	lib2.EXPECT().Wibble2().Return(nil)

	// Run the function we want to test
	err := TryMe2()

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}
