package code

import (
	"testing"
	. "launchpad.net/gocheck"

	"github.com/qur/gomock/gomock"

	"github.com/qur/withmock/scenarios/gocheck/lib" // mock
)

type Base struct{}

var _ = Suite(&Base{})

func Test (t *testing.T) {
	TestingT(t)
}

func (*Base) TestTryMe(c *C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	lib.MOCK().SetController(ctrl)
	lib.EXPECT().Wibble().Return(nil)

	// Run the function we want to test
	err := TryMe()

	// Check error return
	c.Check(err, IsNil)
}
