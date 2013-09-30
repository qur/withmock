package issue8

import (
	"testing"

	"github.com/qur/withmock/scenarios/issue8/bug" // mock

	"code.google.com/p/gomock/gomock"
)

func TestTryMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := make(chan interface{})

	bug.MOCK().SetController(ctrl)

	bug.EXPECT().TryMe(c).Return(nil)

	err := TryMe(c)

	if err != nil {
		t.Errorf("TryMe returned non-nil error: %s", err)
	}
}

func TestTryMe2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := make(chan<- interface{})

	bug.MOCK().SetController(ctrl)

	bug.EXPECT().TryMe2(c).Return(nil)

	err := TryMe2(c)

	if err != nil {
		t.Errorf("TryMe returned non-nil error: %s", err)
	}
}

func TestTryMe3(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := make(<-chan interface{})

	bug.MOCK().SetController(ctrl)

	bug.EXPECT().TryMe3(c).Return(nil)

	err := TryMe3(c)

	if err != nil {
		t.Errorf("TryMe returned non-nil error: %s", err)
	}
}
