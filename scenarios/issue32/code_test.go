package code

import (
	"testing"

	"code.google.com/p/gomock/gomock"

	"time" //mock
)

func TestMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t1 := time.Time{}

	time.MOCK().SetController(ctrl)
	time.EXPECT().Now().Return(t1)
	t1.EXPECT().String().Return("wibble")

	if RunMe() != "Time: wibble" {
		t.Errorf("expected mock time")
	}
}
