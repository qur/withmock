package code

import (
	"net"
	"net/http"
	"time"
)

func RunMe(Addr string) error {
	s := &http.Server{
		Addr: Addr,
	}

	ln, err := net.Listen("tcp", Addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	go s.Serve(ln)

	time.Sleep(2 * time.Second)

	return nil
}
