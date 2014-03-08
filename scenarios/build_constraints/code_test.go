package code

import (
	"testing"

	"os" // mock

	"github.com/qur/gomock/gomock"

	"github.com/qur/withmock/scenarios/build_constraints/lib" // mock
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

	os.MOCK().SetController(ctrl)

	f := &os.File{}
	f.EXPECT().WriteString("Hello").Return(5, nil)

	// Run the function we want to test
	TryMe2(f)
}
