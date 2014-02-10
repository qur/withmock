package code

import (
	"bytes"
)

func RunMe(s string) string {
	b := bytes.NewBuffer([]byte(s))
	return b.String()
}
