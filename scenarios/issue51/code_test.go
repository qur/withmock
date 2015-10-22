package code

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/qur/withmock/scenarios/issue51/lib" // mock
)

func TestTryMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib.MOCK().SetController(ctrl)

	tp := lib.Test{}
	tp_ptr := &tp

	tp_ptr.EXPECT().PointerMethod().Return(nil)

	// Run the function we want to test
	err := TryMe(tp_ptr)

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}
