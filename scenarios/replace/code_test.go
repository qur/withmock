package code

import (
	"testing"

	"github.com/qur/gomock/gomock"

	"github.com/qur/withmock/scenarios/replace/lib"  // mock
	//"github.com/qur/withmock/scenarios/replace/lib2" // mock
	"github.com/qur/withmock/scenarios/replace/lib2" // replace(github.com/qur/withmock/scenarios/replace/lib2_mock)
)

func TestTryMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib.MOCK().SetController(ctrl)

	lib2.Wibble = nil

	// Run the function we want to test
	err := TryMe()

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}
