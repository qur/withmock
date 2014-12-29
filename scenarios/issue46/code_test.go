package code

import (
	"testing"

	"code.google.com/p/gomock/gomock"

	"net/http" //mock
)

func TestMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

    http.MOCK().SetController(ctrl)
    resp := &http.Response{Status: "567 Test"}
    http.EXPECT().Get("http://www.google.com").Return(resp, nil)

    if RunMe() != "567 Test" {
		t.Errorf("expected mock net/http")
    }
}
