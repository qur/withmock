package lib

import (
	"fmt"
	"time"
)

const interval time.Duration = 15*time.Second

func Wibble() error {
	return fmt.Errorf("Not Mocked!")
}
