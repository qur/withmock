package code

import (
	"os"
	"os/signal"
)

func RunMe() os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	return <-c
}
