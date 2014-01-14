package code

import (
	"testing"

	"code.google.com/p/gomock/gomock"

	"github.com/lunastorm/withmock/scenarios/issue29/lib" // mock
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
