package code

import (
	"github.com/qur/withmock/scenarios/ssh/lib"

	"golang.org/x/crypto/ssh"
)

func TryMe() error {
	return lib.Wibble()
}

func Other() string {
	return ssh.KeyAlgoED25519
}
