package code

import (
	"github.com/golang/protobuf/proto"
	"github.com/qur/withmock/scenarios/basic/lib"
)

var _ = proto.WireBytes

func TryMe() error {
	return lib.Wibble()
}
