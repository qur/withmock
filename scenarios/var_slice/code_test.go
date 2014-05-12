package code

import (
	"testing"

	"code.google.com/p/gomock/gomock"

	"github.com/qur/withmock/scenarios/var_slice/lib" // mock
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
