package code

import (
	"testing"

	"code.google.com/p/gomock/gomock"

	"bytes" //mock
)

func TestMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	b := &bytes.Buffer{}

	bytes.MOCK().SetController(ctrl)
	bytes.EXPECT().NewBuffer("foo").Return(b)
	b.EXPECT().String().Return("wibble")

	if RunMe("foo") != "wibble" {
		t.Errorf("expected mock bytes")
	}
}
