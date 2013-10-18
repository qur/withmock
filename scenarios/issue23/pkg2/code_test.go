package code

import (
	"testing"

	"code.google.com/p/gomock/gomock"

	"github.com/qur/withmock/scenarios/issue23/pkg1/lib" // mock
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

func TestTryMe2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Run the function we want to test
	ret := TryMe2()

	if ret != 0 {
		t.Errorf("Unexpected return: %s (expected 0)", ret)
	}
}
