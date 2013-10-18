package code

import (
	"testing"

	"code.google.com/p/gomock/gomock"

	"github.com/qur/withmock/scenarios/issue23/pkg1/lib" // mock
	"github.com/qur/withmock/scenarios/issue23/shared" // mock
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

	shared.MOCK().SetController(ctrl)
	shared.EXPECT().Wibble().Return(5)

	// Run the function we want to test
	ret := TryMe2()

	if ret != 5 {
		t.Errorf("Unexpected return: %s (expected 5)", ret)
	}
}
