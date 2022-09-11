package router

import (
	"context"
	"io"
	"log"

	"github.com/armon/go-radix"

	"github.com/qur/withmock/lib/proxy/api"
)

type PrefixRouter struct {
	def  api.Store
	tree *radix.Tree
}

var _ api.Store = (*PrefixRouter)(nil)

func NewPrefixRouter(def api.Store) *PrefixRouter {
	return &PrefixRouter{
		def:  def,
		tree: radix.New(),
	}
}

func (r *PrefixRouter) List(ctx context.Context, mod string) ([]string, error) {
	return r.getStore(mod).List(ctx, mod)
}

func (r *PrefixRouter) Info(ctx context.Context, mod, ver string) (*api.Info, error) {
	return r.getStore(mod).Info(ctx, mod, ver)
}

func (r *PrefixRouter) ModFile(ctx context.Context, mod, ver string) (io.Reader, error) {
	return r.getStore(mod).ModFile(ctx, mod, ver)
}

func (r *PrefixRouter) Source(ctx context.Context, mod, ver string) (io.Reader, error) {
	return r.getStore(mod).Source(ctx, mod, ver)
}

func (r *PrefixRouter) getStore(mod string) api.Store {
	prefix, data, matched := r.tree.LongestPrefix(mod)
	if !matched {
		log.Printf("ROUTER: match miss (%s)", mod)
		return r.def
	}
	log.Printf("ROUTER: match hit (%s): %s", mod, prefix)
	return data.(api.Store)
}

func (r *PrefixRouter) Add(prefix string, store api.Store) {
	r.tree.Insert(prefix, store)
}
