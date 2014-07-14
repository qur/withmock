package code

import (
	"bufio"
	"fmt"
	"strings"
)

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
