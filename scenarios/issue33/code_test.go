package code

import (
	"testing"

	"github.com/golang/mock/gomock"

	"bytes" //mock
)

func TestMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	b := &bytes.Buffer{}

	bytes.MOCK().SetController(ctrl)
	bytes.EXPECT().NewBuffer([]byte("foo")).Return(b)
	b.EXPECT().String().Return("wibble")

	if RunMe("foo") != "wibble" {
		t.Errorf("expected mock bytes")
	}
}
