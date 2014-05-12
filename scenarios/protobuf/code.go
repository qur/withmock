package code

import (
	"code.google.com/p/goprotobuf/proto"
	"github.com/qur/withmock/scenarios/basic/lib"
)

var _ = proto.WireBytes

func TryMe() error {
	return lib.Wibble()
}
