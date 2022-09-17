package main

import (
	"log"
	"net/http"

	"github.com/qur/withmock/lib/codemod"
	"github.com/qur/withmock/lib/proxy/basic"
	"github.com/qur/withmock/lib/proxy/cache"
	"github.com/qur/withmock/lib/proxy/modify"
	"github.com/qur/withmock/lib/proxy/router"
	"github.com/qur/withmock/lib/proxy/upstream"
	"github.com/qur/withmock/lib/proxy/web"
)

func main() {
	m := codemod.NewDstModifier()

	u := upstream.NewStore("https://proxy.golang.org")
	i := modify.NewInjector(m, "scratch", u)
	r := router.NewPrefixRouter(i)
	c := cache.NewDir("cache", r)
	handler := web.Register(c)

	const prefix = "gowm.in/if/"

	ig := codemod.NewInterfaceGenerator(prefix)
	p := basic.NewPrefixStripper(prefix, u)
	g := modify.NewInterfaceGenerator(ig, "scratch", p)
	r.Add(prefix, g)

	server := &http.Server{
		Addr:    ":4000",
		Handler: handler,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("listen failed: %s", err)
	}

}
