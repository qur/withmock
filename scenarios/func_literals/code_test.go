package code

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/qur/withmock/scenarios/func_literals/lib" // mock
)

func TestShow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib.MOCK().SetController(ctrl)

	lib.EXPECT().Bar().Return(nil)

	// Run the function we want to test
	err := TryMe()

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}
