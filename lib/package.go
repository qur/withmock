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
	Label() string
	Path() string
	Loc() codeLoc
	HasNonGoCode() (bool, error)

	GetImports() (importSet, error)
	MockImports(map[string]string, *Config) error

	Link() (importSet, error)
	Gen(mock bool, cfg *MockConfig) (importSet, error)
	Install() error
}

type realPackage struct {
	name string
	label string
	path string
	src, dst string
	tmpDir string
	tmpPath string
	goPath string
}

func NewPackage(pkgName, label, tmpDir, goPath string) (Package, error) {
	path, err := LookupImportPath(pkgName)
	if err != nil {
		return nil, Cerr{"LookupImportPath", err}
	}

	codeSrc, err := filepath.Abs(path)
	if err != nil {
		return nil, Cerr{"filepath.Abs", err}
	}

	tmpPath := getTmpPath(tmpDir)

	return &realPackage{
		name: pkgName,
		label: label,
		path: path,
		src: codeSrc,
		dst: filepath.Join(tmpPath, "src", label),
		tmpDir: tmpDir,
		tmpPath: tmpPath,
		goPath: goPath,
	}, nil
}

func NewStdlibPackage(pkgName, label, tmpDir, goRoot string) (Package, error) {
	path, err := LookupImportPath(pkgName)
	if err != nil {
		return nil, Cerr{"LookupImportPath", err}
	}

	codeSrc, err := filepath.Abs(path)
	if err != nil {
		return nil, Cerr{"filepath.Abs", err}
	}

	tmpRoot := getTmpRoot(tmpDir)

	return &realPackage{
		name: pkgName,
		label: label,
		path: path,
		src: codeSrc,
		dst: filepath.Join(tmpRoot, "src", "pkg", label),
		tmpDir: tmpDir,
		tmpPath: tmpRoot,
		goPath: goRoot,
	}, nil
}

func (p *realPackage) Name() string {
	return p.name
}

func (p *realPackage) Label() string {
	return p.label
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
		if strings.HasPrefix(env[i], "GOROOT=") {
			env[i] = "__IGNORE="
		}
	}

	// Setup the environment variables that we want
	env = append(env, "GOPATH=" + p.tmpPath)
	env = append(env, "GOROOT=" + getTmpRoot(p.tmpDir))
	//env = append(env, "ORIG_GOPATH=" + c.origPath)

	cmd := exec.Command(command, args...)
	cmd.Env = env
	return cmd
}

func (p *realPackage) needsInstall() (bool, error) {
	d, err := os.Open(p.dst)
	if err != nil {
		return false, Cerr{"os.Open", err}
	}
	defer d.Close()

	files, err := d.Readdirnames(-1)
	if err != nil {
		return false, Cerr{"d.Readdirnames", err}
	}

	for _, name := range files {
		if strings.HasSuffix(name, ".go") {
			return true, nil
		}
	}

	return false, nil
}

func (p *realPackage) Install() error {
	if getMark(p.label) == testMark {
		// we don't install packages marked for test
		return nil
	}

	needsInstall, err := p.needsInstall()
	if err != nil {
		return Cerr{"p.needsInstall", err}
	}

	if !needsInstall {
		return nil
	}

	cmd := p.insideCommand("go", "install", p.label)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to install '%s': %s\noutput:\n%s",
			p.label, err, out)
	}
	return nil
}
