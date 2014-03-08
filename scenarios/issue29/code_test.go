package code

import (
	"testing"

	"github.com/qur/gomock/gomock"

	"github.com/qur/withmock/scenarios/issue29/lib" // mock
)

type Bar struct {
    lib.Adder
}

func TestAdd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib.MOCK().SetController(ctrl)
    foo := lib.Foo{}
    foo.EXPECT().Add(1, 2).Return(0)

    bar := &Bar{&foo}
    res := bar.Add(1, 2)
    if res != 0 {
        t.Errorf("wrong result: %d\n", res)
    }
}

func TestSplitter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lib.MOCK().SetController(ctrl)

	splitter := lib.MOCK().NewSplitter()
	splitter.EXPECT().Split(5).Return(0, 0)

	var s lib.Splitter = splitter

	x, y := s.Split(5)
	if x+y != 0 {
		t.Errorf("expected mock splitter")
	}
}
