package code

import (
	"testing"

	"github.com/golang/mock/gomock"
)

func TestTryMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Run the function we want to test
	TryMe()
}
