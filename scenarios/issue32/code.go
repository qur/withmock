package code

import (
	"fmt"
	"time"
)

func RunMe() string {
	t := time.Now()
	return fmt.Sprintf("Time: %s", t)
}
