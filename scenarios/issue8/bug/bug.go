package bug

import "fmt"

func TryMe(c chan interface{}) error {
	return fmt.Errorf("Not Mocked!")
}

func TryMe2(c chan<- interface{}) error {
	return fmt.Errorf("Not Mocked!")
}

func TryMe3(c <-chan interface{}) error {
	return fmt.Errorf("Not Mocked!")
}
