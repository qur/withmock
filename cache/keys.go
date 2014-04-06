// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cache

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"time"
	"io"
	"log"

	"github.com/qur/withmock/utils"
)

type cacheFileDetails struct {
	Src string `json:"src"`
	Size int64 `json:"size"`
	Mode os.FileMode `json:"mode"`
	ModTime time.Time `json:"mod_time"`
	Hash string `json:"hash"`
}

func getDetails(path string) (cacheFileDetails, error) {
	// TODO: need to include file size, mode, hash etc in key ...

	st, err := os.Stat(path)
	if err != nil {
		return cacheFileDetails{}, utils.Err{"os.Stat", err}
	}

	f, err := os.Open(path)
	if err != nil {
		return cacheFileDetails{}, utils.Err{"os.Open", err}
	}
	defer f.Close()

	h := NewCacheHash()

	log.Printf("START: calcHash")
	if _, err := io.Copy(h, f); err != nil {
		return cacheFileDetails{}, utils.Err{"io.Copy", err}
	}
	hash := hex.EncodeToString(h.Sum(nil))
	log.Printf("END: calcHash")

	return cacheFileDetails{
		Src: path,
		Size: st.Size(),
		Mode: st.Mode(),
		ModTime: st.ModTime(),
		Hash: hash,
	}, nil
}

type CacheFileKey struct {
	Self cacheFileDetails `json:"self"`
	Op string `json:"op"`
	Files []cacheFileDetails `json:"files"`
	hash string
}

func (c *Cache) NewCacheFileKey(op string, srcs ...string) (*CacheFileKey, error) {
	var err error

	files := make([]cacheFileDetails, len(srcs))
	for i, src := range srcs {
		log.Printf("START: getDetails")
		files[i], err = getDetails(src)
		log.Printf("END: getDetails")
		if err != nil {
			return nil, utils.Err{"c.getDetails("+src+")", err}
		}
	}

	return &CacheFileKey{
		Self: c.self,
		Op: op,
		Files: files,
	}, nil
}

func (k *CacheFileKey) Hash() string {
	if k.hash == "" {
		k.calcHash()
	}

	return k.hash
}

func (k *CacheFileKey) calcHash() {
	h := NewCacheHash()

	enc := json.NewEncoder(h)

	if err := enc.Encode(k); err != nil {
		panic("Failed to JSON encode cacheFileKey instance: " + err.Error())
	}

	k.hash = hex.EncodeToString(h.Sum(nil))
}
