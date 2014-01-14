// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"fmt"
	"go/ast"
	"os"
)

type external struct {
	name, impPath, selector string
}

type ifDetails struct {
	methods   []*funcInfo
	locals    []string
	externals []external
}

func (id *ifDetails) addMethod(name string, f *ast.FuncType) []string {
	m := &mockGen{}

	m.collectScopes()

	fi := &funcInfo{
		name:         name,
		realDisabled: true,
	}
	if f.Params != nil {
		for _, param := range f.Params.List {
			field := field{
				expr: m.exprString(param.Type),
			}
			fi.params = append(fi.params, field)
			for i:=1; i<len(param.Names); i++ {
				fi.params = append(fi.params, field)
			}
		}
	}
	if f.Results != nil {
		for _, result := range f.Results.List {
			field := field{
				expr: m.exprString(result.Type),
			}
			fi.results = append(fi.results, field)
			for i:=1; i<len(result.Names); i++ {
				fi.results = append(fi.results, field)
			}
		}
	}
	id.methods = append(id.methods, fi)

	return m.getScopes()
}

func (id *ifDetails) addLocal(name string) {
	id.locals = append(id.locals, name)
}

func (id *ifDetails) addExternal(name, importPath, selector string) {
	id.externals = append(id.externals, external{
		name:     name,
		impPath:  importPath,
		selector: selector,
	})
}

type ifInfo struct {
	filename string
	types    map[string]*ifDetails
	imports  map[string]string
	EXPECT   string
}

func (ii *ifInfo) addImport(name, path string) {
	ii.imports[name] = path
}

func (ii *ifInfo) addType(t *ast.TypeSpec, imports map[string]string) {
	i, ok := t.Type.(*ast.InterfaceType)
	if !ok {
		// Only care about interfaces
		return
	}

	id := &ifDetails{}

	for _, f := range i.Methods.List {
		switch v := f.Type.(type) {
		case *ast.FuncType:
			scopes := id.addMethod(f.Names[0].Name, v)
			for _, scope := range scopes {
				impPath, ok := imports[scope]
				if !ok {
					panic(fmt.Sprintf("Unkown package %s in interface %s",
						scope, t.Name))
				}
				ii.addImport(scope, impPath)
			}
		case *ast.Ident:
			id.addLocal(v.String())
		case *ast.SelectorExpr:
			p, ok := v.X.(*ast.Ident)
			if !ok {
				panic(fmt.Sprintf("Don't know how to handle selector of non"+
					" Ident value: %T", v.X))
			}
			impPath, ok := imports[p.String()]
			if !ok {
				panic(fmt.Sprintf("Unkown package %s in interface %s",
					p, t.Name))
			}
			ii.addImport(p.String(), impPath)
			id.addExternal(p.String(), impPath, v.Sel.String())
		default:
			panic(fmt.Sprintf("Don't expect %T in interface", f.Type))
		}
	}

	ii.types[t.Name.String()] = id
}

type Interfaces map[string]*ifInfo

func newIfInfo(filename string) *ifInfo {
	return &ifInfo{
		filename: filename,
		types:    make(map[string]*ifDetails),
		imports:  make(map[string]string),
	}
}

func (i Interfaces) getMethods(name string, tname string) ([]*funcInfo, error) {
	info := i[name]

	methods := []*funcInfo{}

	t := info.types[tname]

	methods = append(methods, t.methods...)

	for _, n := range t.locals {
		if _, ok := info.types[n]; !ok {
			return nil, fmt.Errorf("Unknown type %s in package %s", n, name)
		}
		m, err := i.getMethods(name, n)
		if err != nil {
			return nil, err
		}
		methods = append(methods, m...)
	}

	for _, e := range t.externals {
		if _, ok := i[e.name]; !ok {
			info, err := loadInterfaceInfo(e.impPath)
			if err != nil {
				return nil, err
			}
			i[e.name] = info
		}

		m, err := i.getMethods(e.name, e.selector)
		if err != nil {
			return nil, err
		}
		methods = append(methods, m...)
	}

	return methods, nil
}

func (i Interfaces) genInterface(name string) error {
	info := i[name]

	out, err := os.Create(info.filename)
	if err != nil {
		return err
	}
	defer out.Close()

	fmt.Fprintf(out, "package %s\n\n", name)
	fmt.Fprintf(out, "import (\n")
	for name, impPath := range info.imports {
		fmt.Fprintf(out, "\t%s \"%s\"\n", name, impPath)
	}
	fmt.Fprintf(out, "\tgomock \"code.google.com/p/gomock/gomock\"\n")
	fmt.Fprintf(out, ")\n\n")
	for tname := range info.types {
		fmt.Fprintf(out, "type Mock%s struct{int}\n", tname)
		fmt.Fprintf(out, "type _mock_%s_rec struct{\n", tname)
		fmt.Fprintf(out, "\tmock *Mock%s\n", tname)
		fmt.Fprintf(out, "}\n\n")

		// Make sure that our mock satisifies the interface
		fmt.Fprintf(out, "var _ %s = &Mock%s{}\n", tname, tname)

		fmt.Fprintf(out, "func (_ *_meta) New%s() *Mock%s {\n", tname, tname)
		fmt.Fprintf(out, "\treturn &Mock%s{}\n", tname)
		fmt.Fprintf(out, "}\n")
		fmt.Fprintf(out, "func (_m *Mock%s) %s() *_mock_%s_rec {\n",
			tname, info.EXPECT, tname)
		fmt.Fprintf(out, "\treturn &_mock_%s_rec{_m}\n", tname)
		fmt.Fprintf(out, "}\n\n")

		methods, err := i.getMethods(name, tname)
		if err != nil {
			return err
		}

		for _, m := range methods {
			m.recv.expr = "*Mock" + tname
			m.writeMock(out)
			m.writeRecorder(out, "_mock_"+tname+"_rec")
		}
	}

	return nil
}

func (i Interfaces) genExtInterface(name string, extPkg string) error {
	info := i[name]

	out, err := os.Create(info.filename)
	if err != nil {
		return err
	}
	defer out.Close()

	fmt.Fprintf(out, "package %s\n\n", name)
	fmt.Fprintf(out, "import (\n")
	fmt.Fprintf(out, "\t. \"%s\"\n", extPkg)
	for name, impPath := range info.imports {
		fmt.Fprintf(out, "\t%s \"%s\"\n", name, impPath)
	}
	fmt.Fprintf(out, "\tgomock \"code.google.com/p/gomock/gomock\"\n")
	fmt.Fprintf(out, ")\n\n")

	fmt.Fprintf(out, "var (\n")
	fmt.Fprintf(out, "\t_ctrl *gomock.Controller\n")
	fmt.Fprintf(out, ")\n\n")

	fmt.Fprintf(out, "func SetController(controller *gomock.Controller) {\n")
	fmt.Fprintf(out, "\t_ctrl = controller\n")
	fmt.Fprintf(out, "}\n")

	for tname := range info.types {
		fmt.Fprintf(out, "type Mock%s struct{int}\n", tname)
		fmt.Fprintf(out, "type _mock_%s_rec struct{\n", tname)
		fmt.Fprintf(out, "\tmock *Mock%s\n", tname)
		fmt.Fprintf(out, "}\n\n")

		// Make sure that our mock satisifies the interface
		fmt.Fprintf(out, "var _ %s = &Mock%s{}\n", tname, tname)

		fmt.Fprintf(out, "func New%s() *Mock%s {\n", tname, tname)
		fmt.Fprintf(out, "\treturn &Mock%s{}\n", tname)
		fmt.Fprintf(out, "}\n")
		fmt.Fprintf(out, "func (_m *Mock%s) %s() *_mock_%s_rec {\n",
			tname, info.EXPECT, tname)
		fmt.Fprintf(out, "\treturn &_mock_%s_rec{_m}\n", tname)
		fmt.Fprintf(out, "}\n\n")

		methods, err := i.getMethods(name, tname)
		if err != nil {
			return err
		}

		for _, m := range methods {
			m.recv.expr = "*Mock" + tname
			m.writeMock(out)
			m.writeRecorder(out, "_mock_"+tname+"_rec")
		}
	}

	return nil
}

func genInterfaces(interfaces Interfaces) error {
	for name, i := range interfaces {
		if i.filename == "" {
			// no filename means this package was only parsed for information,
			// we don't need to write anything out
			continue
		}

		if err := interfaces.genInterface(name); err != nil {
			return err
		}

		// TODO: currently we need to use goimports to add missing imports, we
		// need to sort out our own imports, then we can switch to gofmt.
		if err := fixup(i.filename); err != nil {
			return err
		}
	}

	return nil
}
