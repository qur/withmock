// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cache

import (
	"encoding/hex"
	"encoding/json"

	"github.com/qur/withmock/config"
	"github.com/qur/withmock/utils"
)

type CacheFileKey struct {
	GoRoot    string             `json:"goroot"`
	GoVersion string             `json:"goversion"`
	Self      string             `json:"self"`
	Op        string             `json:"op"`
	Files     []string           `json:"files"`
	Config    *config.MockConfig `json:"config"`
	hash      string
}

func (c *Cache) NewCacheFileKey(op string, srcs ...string) (*CacheFileKey, error) {
	var err error

	files := make([]string, len(srcs))
	for i, src := range srcs {
		files[i], err = c.lookupDetails(src)
		if err != nil {
			return nil, utils.Err{"c.getDetails(" + src + ")", err}
		}
	}

	return &CacheFileKey{
		GoRoot:    c.goRoot,
		GoVersion: c.goVersion,
		Self:      c.self,
		Op:        op,
		Files:     files,
		Config:    c.cfg,
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
