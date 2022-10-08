package main

import (
	"log"
	"net/http"
	"os"

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
	if err := os.Chdir(os.Args[1]); err != nil {
		log.Fatalf("failed to change dir to %s: %s", os.Args[1], err)
	}

	m := codemod.NewDstModifier()

	// built the default route - create modified versions of packages
	u := upstream.NewStore("https://proxy.golang.org")
	ur := router.NewPrefixRouter(u)
	uc := cache.NewDir("cache/input", ur)
	i := modify.NewInjector(m, scratchDir, uc)
	r := router.NewPrefixRouter(i)
	c := cache.NewDir("cache/output", r)
	handler := web.Register(c)

	// add an input route to download the standard library as if it was a module
	// s := stdlib.New("https://go.dev/dl", scratchDir)
	s := stdlib.New("https://electron.quantumfyre.co.uk", scratchDir)
	ur.AddExact("std", s)
	ur.AddExact("gowm.in/std", basic.NewPrefixStripper("gowm.in/", s))

	// add an output route to create interface packages
	const ifPrefix = "gowm.in/if/"
	ig := codemod.NewInterfaceGenerator(ifPrefix)
	ip := basic.NewPrefixStripper(ifPrefix, uc)
	r.Add(ifPrefix, modify.NewSourceGenerator(ig, scratchDir, ip))

	// add an output route to create mock implementations
	const mockPrefix = "gowm.in/mock/"
	mg := mock.NewMockGenerator(mockPrefix, scratchDir, uc)
	mp := basic.NewPrefixStripper(mockPrefix, uc)
	r.Add(mockPrefix, modify.NewSourceGenerator(mg, scratchDir, mp))

	server := &http.Server{
		Addr:    ":4000",
		Handler: handler,
	}

	log.Printf("START SERVER")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("listen failed: %s", err)
	}

}
