package code

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/qur/withmock/scenarios/issue11/dep1" // mock
)

func TestShow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dep1.MOCK().SetController(ctrl)

	foo := &dep1.Foo{}

	dep1.EXPECT().NewFoo().Return(foo)

	foo.EXPECT().Wibble().Return(nil)

	// Run the function we want to test
	err := TryMe()

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}
