package code

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"

	"net" //mock
)

func TestMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	l := net.MOCK().NewListener()
	c := net.MOCK().NewConn()
	e := fmt.Errorf("goodbye")

	net.MOCK().SetController(ctrl)

	gomock.InOrder(
		net.EXPECT().Listen("tcp", ":8080").Return(l, nil),
		l.EXPECT().Accept().Return(c, nil),
		c.EXPECT().Close(),
		l.EXPECT().Accept().Return(nil, e),
		l.EXPECT().Close(),
	)

	if RunMe() != e {
		t.Errorf("expected mock net")
	}
}
