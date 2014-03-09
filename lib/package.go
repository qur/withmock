// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Package interface {
	Name() string
	NewName() string
	Path() string
	Loc() codeLoc
	HasNonGoCode() (bool, error)

	InstallAs(name string)

	GetImports() (importSet, error)
	MockImports(map[string]string, *Config) error

	Link() (importSet, error)
	Gen(mock bool, cfg *MockConfig) (importSet, error)
	Install() error
}

type realPackage struct {
	name string
	newName string
	path string
	src, dst string
	tmpDir string
	tmpPath string
	goPath string
	instName string
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

func (p *realPackage) InstallAs(name string) {
	p.instName = name
}

func (p *realPackage) Path() string {
	return p.path
}

func (p *realPackage) Loc() codeLoc {
	return codeLoc{p.src, p.dst}
}

func (p *realPackage) HasNonGoCode() (bool, error) {
	return hasNonGoCode(p.name)
}

func (p *realPackage) GetImports() (importSet, error) {
	return GetImports(p.path, true)
}

func (p *realPackage) MockImports(importNames map[string]string, cfg *Config) error {
	return MockImports(p.src, p.dst, importNames, cfg)
}

func (p *realPackage) Link() (importSet, error) {
	return LinkPkg(p.goPath, p.tmpPath, p.name)
}

func (p *realPackage) Gen(mock bool, cfg *MockConfig) (importSet, error) {
	return GenPkg(p.goPath, p.tmpPath, p.name, mock, cfg)
}

func (p *realPackage) insideCommand(command string, args ...string) *exec.Cmd {
	env := os.Environ()

	// remove any current GOPATH from the environment
	for i := range env {
		if strings.HasPrefix(env[i], "GOPATH=") {
			env[i] = "__IGNORE="
		}
	}

	// Setup the environment variables that we want
	env = append(env, "GOPATH=" + p.tmpPath)
	//env = append(env, "ORIG_GOPATH=" + c.origPath)

	cmd := exec.Command(command, args...)
	cmd.Env = env
	return cmd
}

func (p *realPackage) Install() error {
	if p.instName == "" {
		return nil
	}

	path := filepath.Join(p.tmpPath, "src", p.instName)

	d, err := os.Open(path)
	if err != nil {
		return Cerr{"os.Open", err}
	}
	defer d.Close()

	files, err := d.Readdirnames(-1)
	if err != nil {
	}

	needsInstall := false
	for _, name := range files {
		if strings.HasSuffix(name, ".go") {
			needsInstall = true
			break
		}
	}

	if !needsInstall {
		return nil
	}

	cmd := p.insideCommand("go", "install", p.instName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to install '%s': %s\noutput:\n%s",
			p.newName, err, out)
	}
	return nil
}
