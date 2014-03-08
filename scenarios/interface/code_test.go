package code_test

import (
	"testing"

	"github.com/qur/gomock/gomock"

	"github.com/qur/withmock/scenarios/interface/lib" // mock

	"github.com/qur/withmock/scenarios/interface"
	"github.com/qur/withmock/scenarios/interface/_mocks_"
)

func TestShow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib.MOCK().SetController(ctrl)

	foo := lib.MOCK().NewFoo()

	foo.EXPECT().Wibble().Return(nil)

	// Run the function we want to test
	err := code.TryMe(foo)

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}

func TestLocalInterface(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	code_mocks.SetController(ctrl)

	toot := code_mocks.NewTooter()

	toot.EXPECT().Toot().Return(nil)

	// Run the function we want to test
	err := code.TryMe2(toot)

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}
