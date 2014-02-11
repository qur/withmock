package code

import (
	"os"
	"testing"

	"code.google.com/p/gomock/gomock"

	"os/signal" //mock
)

func TestMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notify := func(ch chan<- os.Signal, signals ...os.Signal) {
		ch <- os.Kill
	}

	signal.MOCK().SetController(ctrl)
	signal.EXPECT().Notify(gomock.Any(), os.Interrupt).Do(notify)

	if RunMe() != os.Kill {
		t.Errorf("expected mock signal")
	}
}
