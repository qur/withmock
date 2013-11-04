package code

import (
	"testing"

	"code.google.com/p/gomock/gomock"

	"github.com/qur/withmock/scenarios/has_init/lib" // mock
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
	// Disable mocking for lib
	lib.MOCK().MockAll(false)

	// Run the function we want to test
	err := TryMe()

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}

func TestTryMe3(t *testing.T) {
	// Run the function we want to test
	err := TryMe2()

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}
