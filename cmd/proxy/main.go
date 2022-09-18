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
	uc := cache.NewDir("cache/input", u)
	i := modify.NewInjector(m, "scratch", uc)
	r := router.NewPrefixRouter(i)
	c := cache.NewDir("cache/output", r)
	handler := web.Register(c)

	const ifPrefix = "gowm.in/if/"

	ig := codemod.NewInterfaceGenerator(ifPrefix)
	ip := basic.NewPrefixStripper(ifPrefix, uc)
	r.Add(ifPrefix, modify.NewInterfaceGenerator(ig, "scratch", ip))

	const mockPrefix = "gowm.in/mock/"

	mg := codemod.NewMockGenerator(mockPrefix)
	mp := basic.NewPrefixStripper(mockPrefix, uc)
	r.Add(mockPrefix, modify.NewInterfaceGenerator(mg, "scratch", mp))

	server := &http.Server{
		Addr:    ":4000",
		Handler: handler,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("listen failed: %s", err)
	}

}
