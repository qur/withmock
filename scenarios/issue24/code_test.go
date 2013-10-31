package code

import (
    "testing"
    "code.google.com/p/gomock/gomock"
    mockfmt "fmt" // mock
)

func TestPrintIt(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockfmt.MOCK().SetController(ctrl)

    mockfmt.EXPECT().Println("Got a", 10)

    PrintIt(10)
}
