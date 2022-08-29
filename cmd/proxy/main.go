package main

import (
	"log"
	"net/http"

	"github.com/qur/withmock/lib/proxy/cache"
	"github.com/qur/withmock/lib/proxy/upstream"
	"github.com/qur/withmock/lib/proxy/web"
)

func main() {
	u := upstream.NewStore("https://proxy.golang.org")
	c := cache.NewDir("cache", u)
	handler := web.Register(c)

	server := &http.Server{
		Addr:    ":4000",
		Handler: handler,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("listen failed: %s", err)
	}

}
