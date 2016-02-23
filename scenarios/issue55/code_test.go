package withdeps

import (
	"testing"

	"github.com/gin-gonic/gin" // mock

	"github.com/golang/mock/gomock"
)

func TestTryMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	gin.MOCK().SetController(ctrl)

	gin.EXPECT().Mode().Return("mocked")

	// Run the function we want to test
	ret := TryMe()

	if ret != "mocked" {
		t.Errorf("Unexpected return: %s", ret)
	}
}
