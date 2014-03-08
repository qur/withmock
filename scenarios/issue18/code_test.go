package code

import (
	"testing"

	"github.com/qur/gomock/gomock"
)

func TestTryMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Run the function we want to test
	ret := TryMe()

	if ret {
		t.Error("Asm 1: Expected false, got true")
	}
}
