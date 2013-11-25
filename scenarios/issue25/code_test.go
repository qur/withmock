package withdeps

import (
	"testing"

	"code.google.com/p/gcfg" // mock

	"code.google.com/p/gomock/gomock"
)

func TestTryMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	gcfg.MOCK().SetController(ctrl)

	data := "This is some data to decode"
	gcfg.EXPECT().ReadStringInto(gomock.Any(), data).Return(nil)

	// Run the function we want to test
	err := TryMe(data)

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}
