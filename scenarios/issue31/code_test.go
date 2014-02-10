package code

import (
	"testing"

	"github.com/qur/withmock/scenarios/issue31/lib" //mock
)

func TestMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib.MOCK().SetController(ctrl)

	a := lib.MOCK().NewAdder()
	a.EXPECT().Add(1, 2, 3).Return(1)

	if a.Add(1, 2, 3) != 1 {
		t.Errorf("expected mock adder")
	}
}
