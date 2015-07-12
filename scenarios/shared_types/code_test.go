package code

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/qur/withmock/scenarios/shared_types/dep" // mock
)

func TestTryMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dep.MOCK().SetController(ctrl)

	dep.EXPECT().Wibble(1, 2).Return(nil)

	err := TryMe(1, 2)

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}

func TestTryMe2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dep.MOCK().SetController(ctrl)

	dep.EXPECT().Bar(1, 2).Return(0, 0)

	a, b := TryMe2(1, 2)

	if a != 0 {
		t.Errorf("a != 0")
	}

	if b != 0 {
		t.Errorf("b != 0")
	}
}
