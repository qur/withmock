// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"os"
	"time"
)

type cacheFileDetails struct {
	Src string `json:"src"`
	Size int64 `json:"size"`
	Mode os.FileMode `json:"mode"`
	ModTime time.Time `json:"mod_time"`
}

func (c *Cache) getDetails(path string) (cacheFileDetails, error) {
	// TODO: need to include file size, mode, hash etc in key ...

	st, err := os.Stat(path)
	if err != nil {
		return cacheFileDetails{}, Cerr{"os.Stat", err}
	}

	return cacheFileDetails{
		Src: path,
		Size: st.Size(),
		Mode: st.Mode(),
		ModTime: st.ModTime(),
	}, nil
}

type CacheFileKey struct {
	Op string `json:"op"`
	Files []cacheFileDetails `json:"files"`
	hash string
}

func (c *Cache) NewCacheFileKey(op string, srcs ...string) (*CacheFileKey, error) {
	var err error

	files := make([]cacheFileDetails, len(srcs))
	for i, src := range srcs {
		files[i], err = c.getDetails(src)
		if err != nil {
			return nil, Cerr{"c.getDetails("+src+")", err}
		}
	}

	return &CacheFileKey{
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
	h := sha512.New()

	enc := json.NewEncoder(h)

	if err := enc.Encode(k); err != nil {
		panic("Failed to JSON encode cacheFileKey instance: " + err.Error())
	}

	k.hash = hex.EncodeToString(h.Sum(nil))
}
