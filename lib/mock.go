// Copyright 2011 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type field struct {
	names []string
	expr string
}

type funcInfo struct {
	name string
	varidic bool
	recv struct {
		name, expr string
	}
	params, results []field
}

func (fi *funcInfo) IsMethod() bool {
	return fi.recv.expr != ""
}

func (fi *funcInfo) writeReal(out io.Writer) {
	fmt.Fprintf(out, "func ")
	if fi.IsMethod() {
		fmt.Fprintf(out, "(%s %s) ", fi.recv.name, fi.recv.expr)
	}
	fmt.Fprintf(out, "_real_%s(", fi.name)
	for i, param := range fi.params {
		if i > 0 {
			fmt.Fprintf(out, ", ")
		}
		n := strings.Join(param.names, ", ")
		fmt.Fprintf(out, "%s %s", n, param.expr)
	}
	fmt.Fprintf(out, ") ")
	if len(fi.results) > 0 {
		fmt.Fprintf(out, "(")
	}
	for i, result := range fi.results {
		if i > 0 {
			fmt.Fprintf(out, ", ")
		}
		n := strings.Join(result.names, ", ")
		fmt.Fprintf(out, "%s %s", n, result.expr)
	}
	if len(fi.results) > 0 {
		fmt.Fprintf(out, ")")
	}
	fmt.Fprintf(out, " {\n")
	fmt.Fprintf(out, "\tpanic(\"not implemented yet\")\n")
	fmt.Fprintf(out, "}\n")
}

func (fi *funcInfo) writeMock(out io.Writer) {
	fmt.Fprintf(out, "func ")
	if fi.IsMethod() {
		fmt.Fprintf(out, "(_m %s) ", fi.recv.expr)
	}
	fmt.Fprintf(out, "%s(", fi.name)
	for i, param := range fi.params {
		if i > 0 {
			fmt.Fprintf(out, ", ")
		}
		fmt.Fprintf(out, "p%d %s", i, param.expr)
	}
	fmt.Fprintf(out, ") ")
	if len(fi.results) > 0 {
		if len(fi.results) > 1 {
			fmt.Fprintf(out, "(")
		}
		for i, result := range fi.results {
			if i > 0 {
				fmt.Fprintf(out, ", ")
			}
			fmt.Fprintf(out, "%s", result.expr)
		}
		if len(fi.results) > 1 {
			fmt.Fprintf(out, ")")
		}
		fmt.Fprintf(out, " ")
	}
	fmt.Fprintf(out, "{\n")
	if !fi.IsMethod() {
		fmt.Fprintf(out, "\t")
		if len(fi.results) > 0 {
			fmt.Fprintf(out, "return ")
		}
		fmt.Fprintf(out, "pkgMock.%s(", fi.name)
		for i := range fi.params {
			if i > 0 {
				fmt.Fprintf(out, ", ")
			}
			fmt.Fprintf(out, "p%d", i)
		}
		if fi.varidic {
			fmt.Fprintf(out, "...")
		}
		fmt.Fprintf(out, ")\n")
		fmt.Fprintf(out, "}\n")
		fmt.Fprintf(out, "func (_m *packageMock) %s(", fi.name)
		for i, param := range fi.params {
			if i > 0 {
				fmt.Fprintf(out, ", ")
			}
			fmt.Fprintf(out, "p%d %s", i, param.expr)
		}
		fmt.Fprintf(out, ") ")
		if len(fi.results) > 0 {
			if len(fi.results) > 1 {
				fmt.Fprintf(out, "(")
			}
			for i, result := range fi.results {
				if i > 0 {
					fmt.Fprintf(out, ", ")
				}
				fmt.Fprintf(out, "%s", result.expr)
			}
			if len(fi.results) > 1 {
				fmt.Fprintf(out, ")")
			}
			fmt.Fprintf(out, " ")
		}
		fmt.Fprintf(out, "{\n")
	}
	if fi.varidic {
		fmt.Fprintf(out, "\targs := []interface{}{")
		for i := 0; i < len(fi.params)-1; i++ {
			if i > 0 {
				fmt.Fprintf(out, ", ")
			}
			fmt.Fprintf(out, "p%d", i)
		}
		fmt.Fprintf(out, "}\n")
		fmt.Fprintf(out, "\tfor _, v := range p%d {\n", len(fi.params)-1)
		fmt.Fprintf(out, "\t\targs = append(args, v)\n")
		fmt.Fprintf(out, "\t}\n")
		fmt.Fprintf(out, "\t")
		if len(fi.results) > 0 {
			fmt.Fprintf(out, "ret := ")
		}
		fmt.Fprintf(out, "ctrl.Call(_m, \"%s\", args...)\n", fi.name)
	} else {
		fmt.Fprintf(out, "\t")
		if len(fi.results) > 0 {
			fmt.Fprintf(out, "ret := ")
		}
		fmt.Fprintf(out, "ctrl.Call(_m, \"%s\"", fi.name)
		for i := 0; i < len(fi.params); i++ {
			fmt.Fprintf(out, ", p%d", i)
		}
		fmt.Fprintf(out, ")\n")
	}
	for i, result := range fi.results {
		fmt.Fprintf(out, "\tret%d, _ := ret[%d].(%s)\n", i, i, result.expr)
	}
	if len(fi.results) > 0 {
		fmt.Fprintf(out, "\treturn ")
		for i := range fi.results {
			if i > 0 {
				fmt.Fprintf(out, ", ")
			}
			fmt.Fprintf(out, "ret%d", i)
		}
		fmt.Fprintf(out, "\n")
	}
	fmt.Fprintf(out, "}\n")
}

func (fi *funcInfo) writeRecorder(out io.Writer, recorder string) {
	fmt.Fprintf(out, "func (_mr *%s) %s(", recorder, fi.name)
	if fi.varidic {
		// if the method is varidic, there must be at least one
		// argument - so we can code for 1 or more arguments.
		for i := range fi.params {
			if i > 0 {
				fmt.Fprintf(out, "interface{}, ")
			}
			fmt.Fprintf(out, "p%d ", i)
		}
		fmt.Fprintf(out, "...interface{}")
	} else {
		for i := range fi.params {
			if i > 0 {
				fmt.Fprintf(out, ", ")
			}
			fmt.Fprintf(out, "p%d interface{}", i)
		}
	}
	fmt.Fprintf(out, ") *gomock.Call {\n")
	if fi.varidic {
		fmt.Fprintf(out, "\targs := append([]interface{}{")
		for i := 0; i < len(fi.params)-1; i++ {
			if i > 0 {
				fmt.Fprintf(out, ", ")
			}
			fmt.Fprintf(out, "p%d", i)
		}
		fmt.Fprintf(out, "}, p%d...)\n", len(fi.params)-1)
	}
	fmt.Fprintf(out, "\treturn ctrl.RecordCall(_mr.mock, \"%s\"", fi.name)
	if fi.varidic {
		fmt.Fprintf(out, ", args...")
	} else {
		for i := 0; i < len(fi.params); i++ {
			fmt.Fprintf(out, ", p%d", i)
		}
	}
	fmt.Fprintf(out, ")\n")
	fmt.Fprintf(out, "}\n")
}

// MakeMock writes a mock version of the package found at srcPath into dstPath.
// If dstPath already exists, bad things will probably happen.
func MakeMock(srcPath, dstPath string) error {
	isGoFile := func(info os.FileInfo) bool {
		if info.IsDir() {
			return false
		}
		if strings.HasSuffix(info.Name(), "_test.go") {
			return false
		}
		return strings.HasSuffix(info.Name(), ".go")
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, srcPath, isGoFile, parser.ParseComments)
	if err != nil {
		return err
	}

	for name, pkg := range pkgs {
		types := make(map[string]ast.Expr)
		recorders := make(map[string]string)

		for path, file := range pkg.Files {
			filename := filepath.Join(dstPath, filepath.Base(path))

			out, err := os.Create(filename)
			if err != nil {
				return err
			}
			defer out.Close()

			err = mockFile(out, srcPath, file, types, recorders)
			if err != nil {
				return err
			}

			err = fixup(filename)
			if err != nil {
				return err
			}
		}

		filename := filepath.Join(dstPath, name+"_mock.go")

		out, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer out.Close()

		err = mockPkg(out, name, types, recorders)
		if err != nil {
			return err
		}

		err = fixup(filename)
		if err != nil {
			return err
		}
	}

	return nil
}

func literal(name string, types map[string]ast.Expr) (string, bool) {
	exp := types[name]
	switch v := exp.(type) {
	case *ast.SelectorExpr:
		return exprString(exp), true
	case *ast.FuncType:
		return exprString(exp), true
	case *ast.BasicLit:
		return v.Value, false
	case *ast.CompositeLit:
		s := exprString(v.Type) + "{"
		for i := range v.Elts {
			if i > 0 {
				s += ", "
			}
			s += exprString(v.Elts[i])
		}
		s += "}"
		return s, false
	case *ast.StructType, *ast.ArrayType:
		return name + "{}", false
	case *ast.MapType, *ast.ChanType:
		return "make(" + name + ")", false
	case *ast.Ident:
		alias := v.String()
		if _, ok := types[alias]; ok && alias != name {
			return literal(alias, types)
		}
		switch alias {
		case "int", "int8", "int16", "int32", "int64":
			fallthrough
		case "uint", "uint8", "uint16", "uint32", "uint64":
			fallthrough
		case "rune", "byte", "uintptr", "float32", "float64":
			return name + "(0)", false
		case "string":
			return name + "(\"\")", false
		case "bool":
			return "false", false
		case "complex64", "complex128":
			return name + "(0+0i)", false
		default:
			panic(fmt.Sprintf("Can't convert %s:(%v)%T to string in literal",
				name, exp, exp))
		}
	default:
		panic(fmt.Sprintf("Can't convert %s:(%v)%T to string in literal",
			name, exp, exp))
	}
}

func exprString(exp ast.Expr) string {
	switch v := exp.(type) {
	case *ast.BasicLit:
		return v.Value
	case *ast.CompositeLit:
		s := exprString(v.Type) + "{"
		for i := range v.Elts {
			if i > 0 {
				s += ", "
			}
			s += exprString(v.Elts[i])
		}
		s += "}"
		return s
	case *ast.Ident:
		return v.Name
	case *ast.CallExpr:
		s := exprString(v.Fun) + "("
		for i := range v.Args {
			if i > 0 {
				s += ", "
			}
			s += exprString(v.Args[i])
		}
		s += ")"
		return s
	case *ast.Ellipsis:
		if v.Elt == nil {
			return "..."
		} else {
			return "..." + exprString(v.Elt)
		}
	case *ast.ChanType:
		s := ""
		if v.Dir == ast.RECV {
			s += "<-"
		}
		s += "chan"
		if v.Dir == ast.SEND {
			s += "<-"
		}
		s += " " + exprString(v.Value)
		return s
	case *ast.KeyValueExpr:
		return exprString(v.Key) + ": " + exprString(v.Value)
	case *ast.ParenExpr:
		return "(" + exprString(v.X) + ")"
	case *ast.FuncLit:
		// TODO: ...
		return exprString(v.Type) + "{ panic(\"!TODO!\") }"
	case *ast.StarExpr:
		return "*" + exprString(v.X)
	case *ast.SelectorExpr:
		return exprString(v.X) + "." + v.Sel.Name
	case *ast.StructType:
		s := "struct {\n"
		for _, field := range v.Fields.List {
			names := make([]string, 0, len(field.Names))
			for _, ident := range field.Names {
				if ident.IsExported() {
					names = append(names, ident.Name)
				}
			}
			if len(names) == 0 {
				continue
			}
			s += "\t" + strings.Join(names, ", ") + " "
			s += exprString(field.Type) + "\n"
		}
		s += "}"
		return s
	case *ast.ArrayType:
		if v.Len == nil {
			// Slice
			return "[]" + exprString(v.Elt)
		} else {
			// Array
			return "[" + exprString(v.Len) + "]" + exprString(v.Elt)
		}
	case *ast.MapType:
		return "map[" + exprString(v.Key) + "]" + exprString(v.Value)
	case *ast.UnaryExpr:
		return v.Op.String() + exprString(v.X)
	case *ast.InterfaceType:
		if len(v.Methods.List) == 0 {
			return "interface{}"
		} else {
			s := "interface {\n"
			for _, field := range v.Methods.List {
				switch v := field.Type.(type) {
				case *ast.FuncType:
					s += "\t" + field.Names[0].Name
					s += "("
					if v.Params != nil {
						for i, param := range v.Params.List {
							if i > 0 {
								s += ", "
							}
							s += exprString(param.Type)
						}
					}
					s += ")"
					if v.Results != nil {
						s += " "
						if len(v.Results.List) > 1 {
							s += "("
						}
						for i, result := range v.Results.List {
							if i > 0 {
								s += ", "
							}
							s += exprString(result.Type)
						}
						if len(v.Results.List) > 1 {
							s += ")"
						}
					}
				case *ast.SelectorExpr:
					s += exprString(v)
				case *ast.Ident:
					s += exprString(v)
				default:
					panic(fmt.Sprintf("Don't expect %T in interface", field.Type))
				}
				s += "\n"
			}
			s += "}"
			return s
		}
	case *ast.FuncType:
		s := "func("
		if v.Params != nil {
			for i, param := range v.Params.List {
				if i > 0 {
					s += ", "
				}
				s += exprString(param.Type)
			}
		}
		s += ")"
		if v.Results != nil {
			s += " "
			if len(v.Results.List) > 1 {
				s += "("
			}
			for i, result := range v.Results.List {
				if i > 0 {
					s += ", "
				}
				s += exprString(result.Type)
			}
			if len(v.Results.List) > 1 {
				s += ")"
			}
		}
		return s
	case *ast.BinaryExpr:
		return exprString(v.X) + v.Op.String() + exprString(v.Y)
	default:
		panic(fmt.Sprintf("Can't convert (%v)%T to string in exprString", exp, exp))
	}
}

func fixup(filename string) error {
	cmd := exec.Command("goimports", "-w", filename)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to run goimports on '%s': %s\noutput:\n%s",
			filename, err, out)
	}
	return nil
}

func mockPkg(out io.Writer, name string, types map[string]ast.Expr, recorders map[string]string) (err error) {
	fmt.Fprintf(out, "package %s\n\n", name)

	fmt.Fprintf(out, "import \"code.google.com/p/gomock/gomock\"\n\n")

	fmt.Fprintf(out, "type _meta struct{}\n")
	fmt.Fprintf(out, "type packageMock struct{}\n")
	fmt.Fprintf(out, "type _package_Rec struct{\n")
	fmt.Fprintf(out, "\tmock *packageMock\n")
	fmt.Fprintf(out, "}\n\n")

	fmt.Fprintf(out, "var (\n")
	fmt.Fprintf(out, "\tctrl *gomock.Controller\n")
	fmt.Fprintf(out, "\tpkgMock = &packageMock{}\n")
	fmt.Fprintf(out, ")\n\n")

	fmt.Fprintf(out, "func MOCK() *_meta {\n")
	fmt.Fprintf(out, "\treturn nil\n")
	fmt.Fprintf(out, "}\n\n")

	fmt.Fprintf(out, "func (_ *_meta) SetController(_ctrl *gomock.Controller) {\n")
	fmt.Fprintf(out, "\tctrl = _ctrl\n")
	fmt.Fprintf(out, "}\n\n")

	fmt.Fprintf(out, "func EXPECT() *_package_Rec {\n")
	fmt.Fprintf(out, "\treturn &_package_Rec{pkgMock}\n")
	fmt.Fprintf(out, "}\n\n")

	for base, rec := range recorders {
		if _, found := recorders[base[1:]]; base[0] == '*' && found {
			// If pointer and non-pointer receiver, just use the non-pointer
			continue
		}
		name := base
		mod := ""
		if base[0] == '*' {
			name = base[1:]
			mod = "&"
		}
		_, isInterface := types[name].(*ast.InterfaceType)
		if !isInterface && !ast.IsExported(name) {
			lit, cast := literal(name, types)
			if cast {
				fmt.Fprintf(out, "func (_ *_meta) New%s(val %s) %s {\n",
					name, lit, base)
				fmt.Fprintf(out, "\treturn (%s)(%sval)\n", base, mod)
			} else {
				fmt.Fprintf(out, "func (_ *_meta) New%s() %s {\n", name, base)
				fmt.Fprintf(out, "\tval := %s\n", lit)
				fmt.Fprintf(out, "\treturn %sval\n", mod)
			}
			fmt.Fprintf(out, "}\n\n")
		}
		fmt.Fprintf(out, "type %s struct {\n", rec)
		fmt.Fprintf(out, "\tmock %s\n", base)
		fmt.Fprintf(out, "}\n\n")
		fmt.Fprintf(out, "func (_m %s) EXPECT() *%s {\n", base, rec)
		fmt.Fprintf(out, "\treturn &%s{_m}\n", rec)
		fmt.Fprintf(out, "}\n\n")
	}

	return nil
}

var pkgNames = map[string]string{}

func getPackageName(impPath, srcPath string) (string, error) {
	// Special case for the magic "C" package
	if impPath == "C" {
		return "", nil
	}

	name, found := pkgNames[impPath]
	if found {
		return name, nil
	}

	cache := true

	if strings.HasPrefix(impPath, "./") {
		// relative import, no caching, need to change directory
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		defer os.Chdir(cwd)

		os.Chdir(srcPath)
		cache = false
	}

	name, err := GetOutput("go", "list", "-f", "{{.Name}}", impPath)
	if err != nil {
		return "", fmt.Errorf("Failed to get name for '%s': %s", impPath, err)
	}

	if cache {
		pkgNames[impPath] = name
	}

	return name, nil
}

func mockFile(out io.Writer, srcPath string, f *ast.File, types map[string]ast.Expr, recorders map[string]string) (err error) {
	if len(f.Comments) > 0 {
		c := f.Comments[0].Text()
		if strings.HasPrefix(c, "+build") {
			fmt.Fprintf(out, "// %s\n\n", c)
		}
	}

	if f.Doc != nil {
		for _, cmt := range f.Doc.List {
			fmt.Fprintf(out, "%s\n", cmt.Text)
		}
	}

	fmt.Fprintf(out, "package %s\n\n", f.Name)

	fmt.Fprintf(out, "import \"code.google.com/p/gomock/gomock\"\n\n")

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Doc != nil && d.Doc.Text() != "" {
				fmt.Fprintf(out, "/*\n%s*/\n", d.Doc.Text())
			}
			switch d.Tok {
			case token.IMPORT:
				if len(d.Specs) == 1 {
					s := d.Specs[0].(*ast.ImportSpec)
					if s.Doc != nil {
						fmt.Fprintf(out, "%s", s.Doc.Text())
					}
					fmt.Fprintf(out, "import ")
					if s.Name != nil {
						fmt.Fprintf(out, "%s ", s.Name)
					} else {
						impPath := strings.Trim(s.Path.Value, "\"")
						name, err := getPackageName(impPath, srcPath)
						if err != nil {
							return err
						}
						fmt.Fprintf(out, "%s ", name)
					}
					fmt.Fprintf(out, "%s\n\n", s.Path.Value)
					continue
				}
				fmt.Fprintf(out, "import (\n")
				for _, spec := range d.Specs {
					s := spec.(*ast.ImportSpec)
					fmt.Fprintf(out, "\t")
					if s.Name != nil {
						fmt.Fprintf(out, "%s ", s.Name)
					} else {
						impPath := strings.Trim(s.Path.Value, "\"")
						name, err := getPackageName(impPath, srcPath)
						if err != nil {
							return err
						}
						fmt.Fprintf(out, "%s ", name)
					}
					fmt.Fprintf(out, "%s\n", s.Path.Value)
				}
				fmt.Fprintf(out, ")\n\n")
			case token.TYPE:
				// We can't ignore private types, as we might be using them.
				if len(d.Specs) == 1 {
					t := d.Specs[0].(*ast.TypeSpec)
					fmt.Fprintf(out, "type %s %s\n\n", t.Name, exprString(t.Type))
					types[t.Name.String()] = t.Type
				} else {
					fmt.Fprintf(out, "type (\n")
					for i := range d.Specs {
						t := d.Specs[i].(*ast.TypeSpec)
						fmt.Fprintf(out, "\t%s %s\n", t.Name, exprString(t.Type))
						types[t.Name.String()] = t.Type
					}
					fmt.Fprintf(out, ")\n\n")
				}
			case token.VAR:
				fmt.Fprintf(out, "var (\n")
				for _, spec := range d.Specs {
					s := spec.(*ast.ValueSpec)
					names := make([]string, 0, len(s.Names))
					for _, ident := range s.Names {
						if ident.IsExported() {
							names = append(names, ident.Name)
						}
					}
					if len(names) == 0 {
						// Don't care about private variables
						continue
					}
					fmt.Fprintf(out, "\t" + strings.Join(names, ", "))
					if s.Type != nil {
						fmt.Fprintf(out, " %s", exprString(s.Type))
					}
					switch len(s.Values) {
					case 0:
					case 1:
						fmt.Fprintf(out, " = %s", exprString(s.Values[0]))
					default:
						return fmt.Errorf("Multiple values for a var not implemented")
					}
					fmt.Fprintf(out, "\n")
				}
				fmt.Fprintf(out, ")\n\n")
			case token.CONST:
				fmt.Fprintf(out, "const (\n")
				for _, spec := range d.Specs {
					s := spec.(*ast.ValueSpec)
					if len(s.Names) != 1 {
						return fmt.Errorf("Multiple names for a constant not implemented")
					}
					fmt.Fprintf(out, "\t%s", s.Names[0])
					if s.Type != nil {
						fmt.Fprintf(out, " %s", s.Type)
					}
					switch len(s.Values) {
					case 0:
					case 1:
						fmt.Fprintf(out, " = %s", exprString(s.Values[0]))
					default:
						return fmt.Errorf("Multiple values for a constant not implemented")
					}
					fmt.Fprintf(out, "\n")
				}
				fmt.Fprintf(out, ")\n\n")
			default:
				fmt.Fprintf(out, "--- unknown GenDecl Token: %v\n", d.Tok)
			}
		case *ast.FuncDecl:
			if d.Body == nil || !d.Name.IsExported() {
				// ignore forward declarations, and non-exported functions
				continue
			}
			fi := &funcInfo{name: d.Name.String()}
			recorder := "_package_Rec"
			if d.Recv != nil {
				fi.recv.name = d.Recv.List[0].Names[0].String()
				t := exprString(d.Recv.List[0].Type)
				fi.recv.expr = t
				recorder = fmt.Sprintf("_%s_Rec", t)
				if s, ok := d.Recv.List[0].Type.(*ast.StarExpr); ok {
					recorder = fmt.Sprintf("_%s_Rec", exprString(s.X))
				}
				recorders[t] = recorder
			}
			for _, param := range d.Type.Params.List {
				p := field{
					names: make([]string, len(param.Names)),
					expr: exprString(param.Type),
				}
				for i, name := range param.Names {
				    p.names[i] = name.String()
				}
				_, fi.varidic = param.Type.(*ast.Ellipsis)
				fi.params = append(fi.params, p)
			}
			if d.Type.Results != nil {
				for _, result := range d.Type.Results.List {
					r := field{
						names: make([]string, len(result.Names)),
						expr: exprString(result.Type),
					}
					for i, name := range result.Names {
						r.names[i] = name.String()
					}
					fi.results = append(fi.results, r)
				}
			}

			fi.writeReal(out)
			fi.writeMock(out)
			fi.writeRecorder(out, recorder)
			fmt.Fprintf(out, "\n")
		default:
			fmt.Fprintf(out, "--- Unknown Decl Type: %T\n", decl)
		}
	}

	return nil
}
