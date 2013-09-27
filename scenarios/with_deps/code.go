package withdeps

import (
	"bufio"
	"fmt"
	"os"
)

func Show(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

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
