package lib2

import "fmt"

func Bar() error {
	return fmt.Errorf("Not Mocked!")
}
