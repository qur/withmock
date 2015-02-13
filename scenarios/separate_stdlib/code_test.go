package code_test

import (
	"testing"

	"fmt" // mock

	"github.com/qur/withmock/scenarios/separate_stdlib/code"

	"github.com/qur/gomock/gomock"
)

func TestShow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fmt.MOCK().SetController(ctrl)

	// We need some test data
	data := "one\ntwo\nthree"

	// The test data contains three lines that we expect to be printed.
	fmt.EXPECT().Printf("%d: %s\n", 1, "one")
	fmt.EXPECT().Printf("%d: %s\n", 2, "two")
	fmt.EXPECT().Printf("%d: %s\n", 3, "three")

	// Run the function we want to test
	err := code.Show(data)

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}
