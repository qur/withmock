package dep

import "fmt"

func Wibble(a, b int) error {
	return fmt.Errorf("Not Mocked!")
}

func Bar(x, y int) (a, b int) {
	return x, y
}
