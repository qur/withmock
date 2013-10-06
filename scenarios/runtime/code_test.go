package code

import (
	"testing"

	"github.com/qur/withmock/scenarios/runtime/dep"

	"code.google.com/p/gomock/gomock"
)

func TestTryMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dep.MOCK().SetController(ctrl)
	dep.MOCK().MockAll(true)

	dep.EXPECT().Wibble(1, 2).Return(5)

	ret := TryMe(1, 2)

	if ret != 5 {
		t.Errorf("TryMe returned %d, not 5", ret)
	}
}

func Test2TryMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dep.MOCK().SetController(ctrl)
	dep.MOCK().MockAll(false)

	ret := TryMe(1, 2)

	if ret != 3 {
		t.Errorf("TryMe returned %d, not 3", ret)
	}
}

func TestFoo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dep.MOCK().SetController(ctrl)
	dep.MOCK().MockAll(true)

	f := &dep.Foo{}

	dep.EXPECT().NewFoo(1).Return(f)
	f.EXPECT().Wibble(2).Return(5)

	ret := TryMe3(1, 2)

	if ret != 5 {
		t.Errorf("TryMe returned %d, not 5", ret)
	}
}

func TestDisableMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dep.MOCK().SetController(ctrl)
	dep.MOCK().MockAll(true)
	dep.MOCK().DisableMock("NewFoo")
	dep.MOCK().DisableMock("Foo.Wibble")

	dep.EXPECT().Wibble(1, 2).Return(5)

	ret := TryMe(1, 2)

	if ret != 5 {
		t.Errorf("TryMe returned %d, not 5", ret)
	}

	ret = TryMe3(2, 3)

	if ret != 6 {
		t.Errorf("TryMe returned %d, not 6", ret)
	}
}
