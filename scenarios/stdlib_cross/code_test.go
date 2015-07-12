package code

import (
	"testing"

	"github.com/golang/mock/gomock"

	"net"      // mock
	"net/http" // mock(net)
	"time"     // mock
)

func TestMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	l := net.MOCK().NewListener()
	c := net.MOCK().NewConn()

	addr := ":8080"

	net.MOCK().SetController(ctrl)
	http.MOCK().SetController(ctrl)

	gomock.InOrder(
		net.EXPECT().Listen("tcp", addr).Return(l, nil),
		time.EXPECT().Sleep(2 * time.Second),
	)

	if RunMe(addr) != nil {
		t.Errorf("didn't expect error")
	}
}
