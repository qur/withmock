package code

import (
	"testing"

	"code.google.com/p/gomock/gomock"

	"github.com/qur/withmock/scenarios/new_methods/lib" // mock
)

func TestShow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib.MOCK().SetController(ctrl)

	foo := lib.MOCK().NewFoo()

	foo.EXPECT().Wibble().Return(nil)

	// Run the function we want to test
	err := TryMe(foo)

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}
