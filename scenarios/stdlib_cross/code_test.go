package code

import (
	"fmt"
	"testing"

	"github.com/qur/gomock/gomock"

	"net"      // mock
	"net/http" // mock(net)
	"time"     // mock
)

func TestMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	l := net.MOCK().NewListener()
	net.MOCK().NewConn()

	addr := ":8080"

	net.MOCK().SetController(ctrl)
	http.MOCK().SetController(ctrl)
	time.MOCK().SetController(ctrl)

	gomock.InOrder(
		net.EXPECT().Listen("tcp", addr).Return(l, nil),
		time.EXPECT().Sleep(2 * time.Second),
	)

	// These happen in a goroutine, so we may not see them ...
	l.EXPECT().Accept().Return(nil, fmt.Errorf("no thanks.")).AnyTimes()
	l.EXPECT().Close().AnyTimes()

	if RunMe(addr) != nil {
		t.Errorf("didn't expect error")
	}
}
