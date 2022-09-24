package router

import (
	"context"
	"io"
	"log"

	"github.com/armon/go-radix"

	"github.com/qur/withmock/lib/proxy/api"
)

type PrefixRouter struct {
	def   api.Store
	exact map[string]api.Store
	tree  *radix.Tree
}

var _ api.Store = (*PrefixRouter)(nil)

func NewPrefixRouter(def api.Store) *PrefixRouter {
	return &PrefixRouter{
		def:   def,
		exact: map[string]api.Store{},
		tree:  radix.New(),
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
	// check for exact match
	if store, ok := r.exact[mod]; ok {
		log.Printf("ROUTER: match hit (%s): EXACT", mod)
		return store
	}

	// check for longest prefix match
	prefix, data, matched := r.tree.LongestPrefix(mod)
	if matched {
		log.Printf("ROUTER: match hit (%s): %s", mod, prefix)
		return data.(api.Store)
	}

	// fall back to default
	log.Printf("ROUTER: match miss (%s)", mod)
	return r.def
}

func (r *PrefixRouter) Add(prefix string, store api.Store) {
	r.tree.Insert(prefix, store)
}

func (r *PrefixRouter) AddExact(match string, store api.Store) {
	r.exact[match] = store
}
