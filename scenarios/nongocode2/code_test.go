package code

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/qur/withmock/scenarios/basic/lib" // mock
)

func TestTryMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib.MOCK().SetController(ctrl)
	lib.EXPECT().Wibble().Return(nil)

	// Run the function we want to test
	err := TryMe()

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}

func TestTryMe3(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Run the function we want to test
	ret := TryMe3()

	if ret {
		t.Error("TryMe2: Expected false, got true")
	}
}
