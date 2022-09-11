package web

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/qur/withmock/lib/proxy/api"
)

type apiProvider struct {
	s api.Store
}

func (a apiProvider) list(w http.ResponseWriter, r *http.Request) {
	mod := mux.Vars(r)["module"]
	log.Printf("LIST: %s", mod)
	versions, err := a.s.List(r.Context(), mod)
	if ne := api.NotExist(""); errors.As(err, &ne) {
		http.Error(w, err.Error(), http.StatusNotFound)
		log.Printf("DEBUG: unknown module (%s): %s", mod, err)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Printf("ERROR: failed to get list response (%s): %s", mod, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	if _, err := w.Write([]byte(strings.Join(versions, "\n"))); err != nil {
		log.Printf("ERROR: failed to write list response (%s): %s", mod, err)
	}
}

func (a apiProvider) info(w http.ResponseWriter, r *http.Request) {
	mod := mux.Vars(r)["module"]
	ver := mux.Vars(r)["version"]
	log.Printf("INFO: %s %s", mod, ver)
	info, err := a.s.Info(r.Context(), mod, ver)
	if ne := api.NotExist(""); errors.As(err, &ne) {
		http.Error(w, err.Error(), http.StatusNotFound)
		log.Printf("DEBUG: unknown module (%s): %s", mod, err)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Printf("ERROR: failed to get info response (%s): %s", mod, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		log.Printf("ERROR: failed to encode info (%s, %s): %s", mod, ver, err)
	}
}

func (a apiProvider) mod(w http.ResponseWriter, r *http.Request) {
	mod := mux.Vars(r)["module"]
	ver := mux.Vars(r)["version"]
	log.Printf("MOD: %s %s", mod, ver)
	mf, err := a.s.ModFile(r.Context(), mod, ver)
	if ne := api.NotExist(""); errors.As(err, &ne) {
		http.Error(w, err.Error(), http.StatusNotFound)
		log.Printf("DEBUG: unknown module (%s): %s", mod, err)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Printf("ERROR: failed to get modfile response (%s): %s", mod, err)
		return
	}
	if closer, ok := mf.(io.Closer); ok {
		defer closer.Close()
	}
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	if _, err := io.Copy(w, mf); err != nil {
		log.Printf("ERROR: failed to copy modfile (%s, %s): %s", mod, ver, err)
	}

}
func (a apiProvider) zip(w http.ResponseWriter, r *http.Request) {
	mod := mux.Vars(r)["module"]
	ver := mux.Vars(r)["version"]
	log.Printf("ZIP: %s %s", mod, ver)
	src, err := a.s.Source(r.Context(), mod, ver)
	if ne := api.NotExist(""); errors.As(err, &ne) {
		http.Error(w, err.Error(), http.StatusNotFound)
		log.Printf("DEBUG: unknown module (%s): %s", mod, err)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Printf("ERROR: failed to get source response (%s): %s", mod, err)
		return
	}
	if closer, ok := src.(io.Closer); ok {
		defer closer.Close()
	}
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	if _, err := io.Copy(w, src); err != nil {
		log.Printf("ERROR: failed to copy source (%s, %s): %s", mod, ver, err)
	}
}

func Register(s api.Store) http.Handler {
	a := apiProvider{s}

	r := mux.NewRouter()
	r.HandleFunc("/{module:.+}/@v/list", a.list)
	r.HandleFunc("/{module:.+}/@v/v{version}.info", a.info)
	r.HandleFunc("/{module:.+}/@v/v{version}.mod", a.mod)
	r.HandleFunc("/{module:.+}/@v/v{version}.zip", a.zip)

	return r
}
