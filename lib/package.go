// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"path/filepath"
)

type Package interface {
	Name() string
	NewName() string
	Path() string
	Loc() codeLoc

	GetImports() (map[string]bool, error)
	MockImports(map[string]string, *Config) error

	Link() (map[string]bool, error)
	Gen(mock bool, cfg *MockConfig) (map[string]bool, error)
}

type realPackage struct {
	name string
	newName string
	path string
	src, dst string
	tmpDir string
	tmpPath string
	goPath string
}

func NewPackage(pkgName, tmpDir, goPath string) (Package, error) {
	path, err := LookupImportPath(pkgName)
	if err != nil {
		return nil, Cerr{"LookupImportPath", err}
	}

	codeSrc, err := filepath.Abs(path)
	if err != nil {
		return nil, Cerr{"filepath.Abs", err}
	}

	newName := markImport(pkgName, testMark)
	tmpPath := getTmpPath(tmpDir)

	return &realPackage{
		name: pkgName,
		newName: newName,
		path: path,
		src: codeSrc,
		dst: filepath.Join(tmpPath, "src", newName),
		tmpDir: tmpDir,
		tmpPath: tmpPath,
		goPath: goPath,
	}, nil
}

func (p *realPackage) Name() string {
	return p.name
}

func (p *realPackage) NewName() string {
	return p.newName
}

func (p *realPackage) Path() string {
	return p.path
}

func (p *realPackage) Loc() codeLoc {
	return codeLoc{p.src, p.dst}
}

func (p *realPackage) GetImports() (map[string]bool, error) {
	return GetImports(p.path, true)
}

func (p *realPackage) MockImports(importNames map[string]string, cfg *Config) error {
	return MockImports(p.src, p.dst, importNames, cfg)
}

func (p *realPackage) Link() (map[string]bool, error) {
	return LinkPkg(p.goPath, p.tmpPath, p.name)
}

func (p *realPackage) Gen(mock bool, cfg *MockConfig) (map[string]bool, error) {
	return GenPkg(p.goPath, p.tmpPath, p.name, mock, cfg)
}
