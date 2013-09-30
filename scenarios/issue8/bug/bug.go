package bug

import "fmt"

func TryMe(c chan interface{}) error {
	return fmt.Errorf("Not Mocked!")
}
