package main

import (
	"log"
	"net/http"

	"github.com/qur/withmock/lib/codemod"
	"github.com/qur/withmock/lib/codemod/mock"
	"github.com/qur/withmock/lib/proxy/basic"
	"github.com/qur/withmock/lib/proxy/cache"
	"github.com/qur/withmock/lib/proxy/modify"
	"github.com/qur/withmock/lib/proxy/router"
	"github.com/qur/withmock/lib/proxy/stdlib"
	"github.com/qur/withmock/lib/proxy/upstream"
	"github.com/qur/withmock/lib/proxy/web"
)

const scratchDir = "scratch"

func main() {
	m := codemod.NewDstModifier()

	u := upstream.NewStore("https://proxy.golang.org")
	uc := cache.NewDir("cache/input", u)
	i := modify.NewInjector(m, scratchDir, uc)
	r := router.NewPrefixRouter(i)
	c := cache.NewDir("cache/output", r)
	handler := web.Register(c)

	const ifPrefix = "gowm.in/if/"

	ig := codemod.NewInterfaceGenerator(ifPrefix)
	ip := basic.NewPrefixStripper(ifPrefix, uc)
	r.Add(ifPrefix, modify.NewInterfaceGenerator(ig, scratchDir, ip))

	const mockPrefix = "gowm.in/mock/"

	mg := mock.NewMockGenerator(mockPrefix, scratchDir, uc)
	mp := basic.NewPrefixStripper(mockPrefix, uc)
	r.Add(mockPrefix, modify.NewInterfaceGenerator(mg, scratchDir, mp))

	const mockStdPrefix = "gowm.in/mock/std/"

	s := stdlib.New(scratchDir)
	sc := cache.NewDir("cache/stdlib", s)
	sg := mock.NewMockGenerator(mockStdPrefix, scratchDir, sc)
	sp := basic.NewPrefixStripper(mockStdPrefix, sc)
	r.Add(mockStdPrefix, modify.NewInterfaceGenerator(sg, scratchDir, sp))

	server := &http.Server{
		Addr:    ":4000",
		Handler: handler,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("listen failed: %s", err)
	}

}
