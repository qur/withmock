package withdeps

import (
	"bufio"
	"encoding/json"
	"encoding/base64"
	"fmt"
	"strings"
)

var _ = base64.StdEncoding

func Show(data string) error {
	f := strings.NewReader(data)

	s := bufio.NewScanner(f)
	for i := 1; s.Scan(); i++ {
		line := s.Text()
		fmt.Printf("%d: %s\n", i, line)
	}
	if err := s.Err(); err != nil {
		return err
	}

	return nil
}

type s struct {
	s string
}

func Show2(data string) ([]byte, error) {
	return json.Marshal(&s{data})
}
