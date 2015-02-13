package code

import (
	"testing"

	"github.com/qur/gomock/gomock"

	"github.com/qur/withmock/scenarios/separate/code"
	"github.com/qur/withmock/scenarios/separate/lib" // mock
)

func TestTryMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib.MOCK().SetController(ctrl)
	lib.EXPECT().Wibble().Return(nil)

	// Run the function we want to test
	err := code.TryMe()

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}
