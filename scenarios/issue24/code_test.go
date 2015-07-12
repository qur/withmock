package code

import (
    "testing"
    "github.com/golang/mock/gomock"
    mockfmt "fmt" // mock
)

func TestPrintIt(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockfmt.MOCK().SetController(ctrl)

    mockfmt.EXPECT().Println("Got a", 10)

    PrintIt(10)
}
