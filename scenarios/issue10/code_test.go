package withdeps

import (
	"testing"

	"github.com/qur/gomock/gomock"

	"github.com/qur/withmock/scenarios/issue10/dep1" // mock
)

func TestShow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// We need some test data
	data := "one\ntwo\nthree"

	dep1.MOCK().SetController(ctrl)

	dep1.EXPECT().Modify(data).Return(nil)

	// Run the function we want to test
	err := Show(data)

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}
