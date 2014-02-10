package code

import (
	"testing"

	"code.google.com/p/gomock/gomock"

	"time" //mock
)

func TestMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t := time.Time{}

	time.MOCK().SetController(ctrl)
	time.EXPECT().Now().Return(t)
	t.EXPECT().String().Return("wibble")

	if RunMe() != "Time: wibble" {
		t.Errorf("expected mock time")
	}
}
