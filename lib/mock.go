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
	realDisabled bool
	recv struct {
		name, expr string
	}
	params, results []field
	body []byte
}

func (fi *funcInfo) IsMethod() bool {
	return fi.recv.expr != ""
}

func (fi *funcInfo) writeReal(out io.Writer) {
	fmt.Fprintf(out, "func ")
	if fi.IsMethod() {
		fmt.Fprintf(out, "(%s %s) ", fi.recv.name, fi.recv.expr)
	}
	if ast.IsExported(fi.name) {
		fmt.Fprintf(out, "_real_")
	}
	fmt.Fprintf(out, "%s(", fi.name)
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
		for i, result := range fi.results {
			if i > 0 {
				fmt.Fprintf(out, ", ")
			}
			n := strings.Join(result.names, ", ")
			fmt.Fprintf(out, "%s %s", n, result.expr)
		}
		fmt.Fprintf(out, ") ")
	}
	out.Write(fi.body)
	fmt.Fprintf(out, "\n")
}

func (fi *funcInfo) countParams() int {
	p := 0
	for _, param := range fi.params {
		if len(param.names) == 0 {
			p++
		} else {
			p += len(param.names)
		}
	}
	return p
}

func (fi *funcInfo) writeParams(out io.Writer) int {
	p := 0
	for i, param := range fi.params {
		if i > 0 {
			fmt.Fprintf(out, ", ")
		}
		if len(param.names) == 0 {
			fmt.Fprintf(out, "p%d", p)
			p++
		} else {
			for j := range param.names {
				if j > 0 {
					fmt.Fprintf(out, ", ")
				}
				fmt.Fprintf(out, "p%d", p)
				p++
			}
		}
		fmt.Fprintf(out, " %s", param.expr)
	}
	return p
}

func (fi *funcInfo) retTypes() []string {
	results := make([]string, 0, len(fi.results))
	for _, result := range fi.results {
		x := len(result.names)
		if x == 0 {
			x = 1
		}
		for i := 0; i < x; i++ {
			results = append(results, result.expr)
		}
	}
	return results
}

func (fi *funcInfo) writeMock(out io.Writer) {
	scopedName := fi.name
	fmt.Fprintf(out, "func ")
	if fi.IsMethod() {
		fmt.Fprintf(out, "(_m %s) ", fi.recv.expr)
		if fi.recv.expr[0] == '*' {
			scopedName = fi.recv.expr[1:] + "." + scopedName
		} else {
			scopedName = fi.recv.expr + "." + scopedName
		}
	}
	fmt.Fprintf(out, "%s(", fi.name)
	args := fi.writeParams(out)
	fmt.Fprintf(out, ") ")
	returns := fi.retTypes()
	if len(returns) > 0 {
		fmt.Fprintf(out, "(%s) ", strings.Join(returns, ", "))
	}
	fmt.Fprintf(out, "{\n")
	if !fi.IsMethod() {
		fmt.Fprintf(out, "\t")
		if len(fi.results) > 0 {
			fmt.Fprintf(out, "return ")
		}
		fmt.Fprintf(out, "_pkgMock.%s(", fi.name)
		for i := 0; i < args; i++ {
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
		fmt.Fprintf(out, "func (_m *_packageMock) %s(", fi.name)
		fi.writeParams(out)
		fmt.Fprintf(out, ") ")
		if len(returns) > 0 {
			fmt.Fprintf(out, "(%s) ", strings.Join(returns, ", "))
		}
		fmt.Fprintf(out, "{\n")
	}
	if fi.varidic {
		if !fi.realDisabled {
			fmt.Fprintf(out, "\tif (!_allMocked && !_enabledMocks[\"%s\"]) " +
				"|| _disabledMocks[\"%s\"] {\n", scopedName, scopedName)
			fmt.Fprintf(out, "\t\t")
			if len(fi.results) > 0 {
				fmt.Fprintf(out, "return ")
			}
			if fi.IsMethod() {
				fmt.Fprintf(out, "_m.")
			}
			fmt.Fprintf(out, "_real_%s(", fi.name)
			for i := 0; i < args-1; i++ {
				fmt.Fprintf(out, "p%d, ", i)
			}
			fmt.Fprintf(out, "p%d...", args-1)
			fmt.Fprintf(out, ")\n")
			if len(fi.results) == 0 {
				fmt.Fprintf(out, "\treturn")
			}
			fmt.Fprintf(out, "\t}\n")
		}
		fmt.Fprintf(out, "\targs := []interface{}{")
		for i := 0; i < args-1; i++ {
			if i > 0 {
				fmt.Fprintf(out, ", ")
			}
			fmt.Fprintf(out, "p%d", i)
		}
		fmt.Fprintf(out, "}\n")
		fmt.Fprintf(out, "\tfor _, v := range p%d {\n", args-1)
		fmt.Fprintf(out, "\t\targs = append(args, v)\n")
		fmt.Fprintf(out, "\t}\n")
		fmt.Fprintf(out, "\t")
		if len(fi.results) > 0 {
			fmt.Fprintf(out, "ret := ")
		}
		fmt.Fprintf(out, "_ctrl.Call(_m, \"%s\", args...)\n", fi.name)
	} else {
		if !fi.realDisabled {
			fmt.Fprintf(out, "\tif (!_allMocked && !_enabledMocks[\"%s\"]) " +
				"||  _disabledMocks[\"%s\"] {\n", scopedName, scopedName)
			fmt.Fprintf(out, "\t\t")
			if len(fi.results) > 0 {
				fmt.Fprintf(out, "return ")
			}
			if fi.IsMethod() {
				fmt.Fprintf(out, "_m.")
			}
			fmt.Fprintf(out, "_real_%s(", fi.name)
			for i := 0; i < args; i++ {
				if i > 0 {
					fmt.Fprintf(out, ", ")
				}
				fmt.Fprintf(out, "p%d", i)
			}
			fmt.Fprintf(out, ")\n")
			if len(fi.results) == 0 {
				fmt.Fprintf(out, "\treturn")
			}
			fmt.Fprintf(out, "\t}\n")
		}
		fmt.Fprintf(out, "\t")
		if len(fi.results) > 0 {
			fmt.Fprintf(out, "ret := ")
		}
		fmt.Fprintf(out, "_ctrl.Call(_m, \"%s\"", fi.name)
		for i := 0; i < args; i++ {
			fmt.Fprintf(out, ", p%d", i)
		}
		fmt.Fprintf(out, ")\n")
	}
	for i, ret := range returns {
		fmt.Fprintf(out, "\tret%d, _ := ret[%d].(%s)\n", i, i, ret)
	}
	if len(returns) > 0 {
		fmt.Fprintf(out, "\treturn ")
		for i := 0; i < len(returns); i++ {
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
	args := fi.countParams()
	fmt.Fprintf(out, "func (_mr *%s) %s(", recorder, fi.name)
	if args > 0 {
		if fi.varidic {
			if args > 1 {
				for i := 0; i < args-1; i++ {
					if i > 0 {
						fmt.Fprintf(out, ", ")
					}
					fmt.Fprintf(out, "p%d", i)
				}
				fmt.Fprintf(out, " interface{}, ")
			}
			fmt.Fprintf(out, "p%d ...interface{}", args-1)
		} else {
			for i := 0; i < args; i++ {
				if i > 0 {
					fmt.Fprintf(out, ", ")
				}
				fmt.Fprintf(out, "p%d", i)
			}
			fmt.Fprintf(out, " interface{}")
		}
	}
	fmt.Fprintf(out, ") *gomock.Call {\n")
	if fi.varidic {
		fmt.Fprintf(out, "\targs := append([]interface{}{")
		for i := 0; i < args-1; i++ {
			if i > 0 {
				fmt.Fprintf(out, ", ")
			}
			fmt.Fprintf(out, "p%d", i)
		}
		fmt.Fprintf(out, "}, p%d...)\n", args-1)
	}
	fmt.Fprintf(out, "\treturn _ctrl.RecordCall(_mr.mock, \"%s\"", fi.name)
	if fi.varidic {
		fmt.Fprintf(out, ", args...")
	} else {
		for i := 0; i < args; i++ {
			fmt.Fprintf(out, ", p%d", i)
		}
	}
	fmt.Fprintf(out, ")\n")
	fmt.Fprintf(out, "}\n")
}

type mockGen struct {
	fset *token.FileSet
	srcPath string
	mockByDefault bool
	types map[string]ast.Expr
	recorders map[string]string
	inits []string
	data io.ReaderAt
	ifInfo *ifInfo
}

// MakeMock writes a mock version of the package found at srcPath into dstPath.
// If dstPath already exists, bad things will probably happen.
func MakePkg(srcPath, dstPath string, mock bool) error {
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

	interfaces := make(Interfaces)

	for name, pkg := range pkgs {
		m := &mockGen{
			fset: fset,
			srcPath: srcPath,
			mockByDefault: mock,
			types: make(map[string]ast.Expr),
			recorders: make(map[string]string),
			ifInfo: newIfInfo(filepath.Join(dstPath, name+"_ifmocks.go")),
		}

		for path, file := range pkg.Files {
			srcFile := filepath.Join(srcPath, filepath.Base(path))
			filename := filepath.Join(dstPath, filepath.Base(path))

			out, err := os.Create(filename)
			if err != nil {
				return err
			}
			defer out.Close()

			err = m.file(out, file, srcFile)
			if err != nil {
				return err
			}

			/*
			// TODO: we want to gofmt, goimports can break things ...
			err = fixup(filename)
			if err != nil {
				return err
			}
			*/
		}

		filename := filepath.Join(dstPath, name+"_mock.go")

		out, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer out.Close()

		err = m.pkg(out, name)
		if err != nil {
			return err
		}

		// TODO: currently we need to use goimports to add missing imports, we
		// need to sort out our own imports, then we can switch to gofmt.
		err = fixup(filename)
		if err != nil {
			return err
		}

		interfaces[name] = m.ifInfo
	}

	if err := genInterfaces(interfaces); err != nil {
		return err
	}

	return nil
}

func (m *mockGen) literal(name string) (string, bool) {
	exp := m.types[name]
	switch v := exp.(type) {
	case *ast.SelectorExpr:
		return m.exprString(exp), true
	case *ast.FuncType:
		return m.exprString(exp), true
	case *ast.BasicLit:
		return v.Value, false
	case *ast.CompositeLit:
		s := m.exprString(v.Type) + "{"
		for i := range v.Elts {
			if i > 0 {
				s += ", "
			}
			s += m.exprString(v.Elts[i])
		}
		s += "}"
		return s, false
	case *ast.StructType, *ast.ArrayType:
		return name + "{}", false
	case *ast.MapType, *ast.ChanType:
		return "make(" + name + ")", false
	case *ast.Ident:
		alias := v.String()
		if _, ok := m.types[alias]; ok && alias != name {
			return m.literal(alias)
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

func (m *mockGen) exprString(exp ast.Expr) string {
	switch v := exp.(type) {
	case *ast.BasicLit:
		return v.Value
	case *ast.CompositeLit:
		s := ""
		if v.Type != nil {
			s += m.exprString(v.Type)
		}
		s += "{"
		for i := range v.Elts {
			if i > 0 {
				s += ", "
			}
			s += m.exprString(v.Elts[i])
		}
		s += "}"
		return s
	case *ast.Ident:
		return v.Name
	case *ast.CallExpr:
		s := m.exprString(v.Fun) + "("
		for i := range v.Args {
			if i > 0 {
				s += ", "
			}
			s += m.exprString(v.Args[i])
		}
		s += ")"
		return s
	case *ast.Ellipsis:
		if v.Elt == nil {
			return "..."
		} else {
			return "..." + m.exprString(v.Elt)
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
		s += " " + m.exprString(v.Value)
		return s
	case *ast.KeyValueExpr:
		return m.exprString(v.Key) + ": " + m.exprString(v.Value)
	case *ast.ParenExpr:
		return "(" + m.exprString(v.X) + ")"
	case *ast.FuncLit:
		pos1 := m.fset.Position(v.Body.Lbrace)
		pos2 := m.fset.Position(v.Body.Rbrace)
		body := make([]byte, pos2.Offset-pos1.Offset+1)
		_, err := m.data.ReadAt(body, int64(pos1.Offset))
		if err != nil {
			panic(fmt.Sprintf("Failed to read from m.data: %s", err))
		}
		return m.exprString(v.Type) + " " + string(body)
	case *ast.StarExpr:
		return "*" + m.exprString(v.X)
	case *ast.SelectorExpr:
		return m.exprString(v.X) + "." + v.Sel.Name
	case *ast.StructType:
		s := "struct {\n"
		for _, field := range v.Fields.List {
			names := make([]string, 0, len(field.Names))
			for _, ident := range field.Names {
				names = append(names, ident.Name)
			}
			s += "\t" + strings.Join(names, ", ") + " "
			s += m.exprString(field.Type) + "\n"
		}
		s += "}"
		return s
	case *ast.ArrayType:
		if v.Len == nil {
			// Slice
			return "[]" + m.exprString(v.Elt)
		} else {
			// Array
			return "[" + m.exprString(v.Len) + "]" + m.exprString(v.Elt)
		}
	case *ast.MapType:
		return "map[" + m.exprString(v.Key) + "]" + m.exprString(v.Value)
	case *ast.UnaryExpr:
		return v.Op.String() + m.exprString(v.X)
	case *ast.TypeAssertExpr:
		s := m.exprString(v.X) + ".("
		if v.Type == nil {
			s += "type"
		} else {
			s += m.exprString(v.Type)
		}
		s += ")"
		return s
	case *ast.IndexExpr:
		return m.exprString(v.X) + "[" + m.exprString(v.Index) + "]"
	case *ast.InterfaceType:
		if len(v.Methods.List) == 0 {
			return "interface{}"
		} else {
			s := "interface {\n"
			for _, field := range v.Methods.List {
				s += "\t"
				switch v := field.Type.(type) {
				case *ast.FuncType:
					s += field.Names[0].Name + "("
					if v.Params != nil {
						for i, param := range v.Params.List {
							if i > 0 {
								s += ", "
							}
							s += m.exprString(param.Type)
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
							s += m.exprString(result.Type)
						}
						if len(v.Results.List) > 1 {
							s += ")"
						}
					}
				case *ast.SelectorExpr:
					s += m.exprString(v)
				case *ast.Ident:
					s += m.exprString(v)
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
				s += m.exprString(param.Type)
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
				s += m.exprString(result.Type)
			}
			if len(v.Results.List) > 1 {
				s += ")"
			}
		}
		return s
	case *ast.BinaryExpr:
		return m.exprString(v.X) + v.Op.String() + m.exprString(v.Y)
	default:
		panic(fmt.Sprintf("Can't convert (%v)%T to string in exprString", exp, exp))
	}
}

func fixup(filename string) error {
	cmd := exec.Command("goimports", "-w", filename)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to run gofmt on '%s': %s\noutput:\n%s",
			filename, err, out)
	}
	return nil
}

func (m *mockGen) pkg(out io.Writer, name string) error {
	fmt.Fprintf(out, "package %s\n\n", name)

	fmt.Fprintf(out, "import \"code.google.com/p/gomock/gomock\"\n\n")

	fmt.Fprintf(out, "type _meta struct{}\n")
	fmt.Fprintf(out, "type _packageMock struct{}\n")
	fmt.Fprintf(out, "type _package_Rec struct{\n")
	fmt.Fprintf(out, "\tmock *_packageMock\n")
	fmt.Fprintf(out, "}\n\n")

	fmt.Fprintf(out, "var (\n")
	fmt.Fprintf(out, "\t_allMocked = false\n")
	fmt.Fprintf(out, "\t_enabledMocks = make(map[string]bool)\n")
	fmt.Fprintf(out, "\t_disabledMocks = make(map[string]bool)\n")
	fmt.Fprintf(out, "\t_ctrl *gomock.Controller\n")
	fmt.Fprintf(out, "\t_pkgMock = &_packageMock{}\n")
	fmt.Fprintf(out, ")\n\n")

	fmt.Fprintf(out, "func init() {\n")
	for _, init := range m.inits {
		fmt.Fprintf(out, "\t%s()\n", init)
	}
	fmt.Fprintf(out, "\t_allMocked = %v\n", m.mockByDefault)
	fmt.Fprintf(out, "}\n\n")

	fmt.Fprintf(out, "func MOCK() *_meta {\n")
	fmt.Fprintf(out, "\treturn nil\n")
	fmt.Fprintf(out, "}\n")

	fmt.Fprintf(out, "func (_ *_meta) SetController(controller *gomock.Controller) {\n")
	fmt.Fprintf(out, "\t_ctrl = controller\n")
	fmt.Fprintf(out, "}\n")

	fmt.Fprintf(out, "func (_ *_meta) MockAll(enabled bool) {\n")
	fmt.Fprintf(out, "\t_allMocked = enabled\n")
	fmt.Fprintf(out, "\t_enabledMocks = make(map[string]bool)\n")
	fmt.Fprintf(out, "\t_disabledMocks = make(map[string]bool)\n")
	fmt.Fprintf(out, "}\n")

	fmt.Fprintf(out, "func (_ *_meta) EnableMock(names ...string) {\n")
	fmt.Fprintf(out, "\tfor _, name := range names {\n")
	fmt.Fprintf(out, "\t\t_enabledMocks[name] = true\n")
	fmt.Fprintf(out, "\t\tdelete(_disabledMocks, name)\n")
	fmt.Fprintf(out, "\t}\n")
	fmt.Fprintf(out, "}\n\n")

	fmt.Fprintf(out, "func (_ *_meta) DisableMock(names ...string) {\n")
	fmt.Fprintf(out, "\tfor _, name := range names {\n")
	fmt.Fprintf(out, "\t\t_disabledMocks[name] = true\n")
	fmt.Fprintf(out, "\t\tdelete(_enabledMocks, name)\n")
	fmt.Fprintf(out, "\t}\n")
	fmt.Fprintf(out, "}\n\n")

	fmt.Fprintf(out, "func EXPECT() *_package_Rec {\n")
	fmt.Fprintf(out, "\treturn &_package_Rec{_pkgMock}\n")
	fmt.Fprintf(out, "}\n\n")

	for base, rec := range m.recorders {
		if _, found := m.recorders[base[1:]]; base[0] == '*' && found {
			// If pointer and non-pointer receiver, just use the non-pointer
			continue
		}
		name := base
		mock := "Mock_" + name
		retType := mock
		mod := ""
		if base[0] == '*' {
			name = base[1:]
			mock = "Mock_" + name
			retType = "*" + mock
			mod = "&"
		}
		_, isInterface := m.types[name].(*ast.InterfaceType)
		if !isInterface && !ast.IsExported(name) {
			fmt.Fprintf(out, "type %s struct {\n", mock)
			fmt.Fprintf(out, "\t%s\n", name)
			fmt.Fprintf(out, "}\n")
			lit, cast := m.literal(name)
			if cast {
				fmt.Fprintf(out, "func (_ *_meta) New%s(val %s) %s {\n",
					name, lit, retType)
				fmt.Fprintf(out, "\treturn %s%s{(%s)(val)}\n", mod, mock, base)
			} else {
				fmt.Fprintf(out, "func (_ *_meta) New%s() %s {\n", name,
					retType)
				fmt.Fprintf(out, "\treturn %s%s{%s}\n", mod, mock, lit)
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

func (m *mockGen) file(out io.Writer, f *ast.File, filename string) error {
	data, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer data.Close()

	// Make sure data is available to exprString/literal
	m.data = data

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

	imports := make(map[string]string)

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
					impPath := strings.Trim(s.Path.Value, "\"")
					if impPath == "code.google.com/p/gomock/gomock" {
						continue
					}
					if s.Doc != nil {
						fmt.Fprintf(out, "%s", s.Doc.Text())
					}
					fmt.Fprintf(out, "import ")
					if s.Name != nil {
						fmt.Fprintf(out, "%s ", s.Name)
						imports[s.Name.String()] = impPath
					} else {
						name, err := getPackageName(impPath, m.srcPath)
						if err != nil {
							return err
						}
						fmt.Fprintf(out, "%s ", name)
						imports[name] = impPath
					}
					fmt.Fprintf(out, "%s\n\n", s.Path.Value)
					continue
				}
				fmt.Fprintf(out, "import (\n")
				for _, spec := range d.Specs {
					s := spec.(*ast.ImportSpec)
					impPath := strings.Trim(s.Path.Value, "\"")
					if impPath == "code.google.com/p/gomock/gomock" {
						continue
					}
					fmt.Fprintf(out, "\t")
					if s.Name != nil {
						fmt.Fprintf(out, "%s ", s.Name)
						imports[s.Name.String()] = impPath
					} else {
						name, err := getPackageName(impPath, m.srcPath)
						if err != nil {
							return err
						}
						fmt.Fprintf(out, "%s ", name)
						imports[name] = impPath
					}
					fmt.Fprintf(out, "%s\n", s.Path.Value)
				}
				fmt.Fprintf(out, ")\n\n")
			case token.TYPE:
				// We can't ignore private types, as we might be using them.
				if len(d.Specs) == 1 {
					t := d.Specs[0].(*ast.TypeSpec)
					fmt.Fprintf(out, "type %s %s\n\n", t.Name, m.exprString(t.Type))
					m.types[t.Name.String()] = t.Type
					m.ifInfo.addType(t, imports)
				} else {
					fmt.Fprintf(out, "type (\n")
					for i := range d.Specs {
						t := d.Specs[i].(*ast.TypeSpec)
						fmt.Fprintf(out, "\t%s %s\n", t.Name, m.exprString(t.Type))
						m.types[t.Name.String()] = t.Type
						m.ifInfo.addType(t, imports)
					}
					fmt.Fprintf(out, ")\n\n")
				}
			case token.VAR:
				fmt.Fprintf(out, "var (\n")
				for _, spec := range d.Specs {
					s := spec.(*ast.ValueSpec)
					names := make([]string, 0, len(s.Names))
					for _, ident := range s.Names {
						names = append(names, ident.Name)
					}
					fmt.Fprintf(out, "\t" + strings.Join(names, ", "))
					if s.Type != nil {
						fmt.Fprintf(out, " %s", m.exprString(s.Type))
					}
					switch len(s.Values) {
					case 0:
					case 1:
						fmt.Fprintf(out, " = %s", m.exprString(s.Values[0]))
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
						fmt.Fprintf(out, " = %s", m.exprString(s.Values[0]))
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
			fi := &funcInfo{name: d.Name.String()}
			recorder := "_package_Rec"
			if d.Recv != nil {
				if len(d.Recv.List[0].Names) > 0 {
					fi.recv.name = d.Recv.List[0].Names[0].String()
				}
				t := m.exprString(d.Recv.List[0].Type)
				fi.recv.expr = t
				recorder = fmt.Sprintf("_%s_Rec", t)
				if s, ok := d.Recv.List[0].Type.(*ast.StarExpr); ok {
					recorder = fmt.Sprintf("_%s_Rec", m.exprString(s.X))
				}
				m.recorders[t] = recorder
			}
			for _, param := range d.Type.Params.List {
				p := field{
					names: make([]string, len(param.Names)),
					expr: m.exprString(param.Type),
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
						expr: m.exprString(result.Type),
					}
					for i, name := range result.Names {
						r.names[i] = name.String()
					}
					fi.results = append(fi.results, r)
				}
			}
			if d.Body != nil {
				pos1 := m.fset.Position(d.Body.Lbrace)
				pos2 := m.fset.Position(d.Body.Rbrace)
				fi.body = make([]byte, pos2.Offset-pos1.Offset+1)
				_, err := data.ReadAt(fi.body, int64(pos1.Offset))
				if err != nil {
					return err
				}
			}

			if fi.name == "init" && !fi.IsMethod() {
				fi.name = fmt.Sprintf("_real_init_%d", len(m.inits))
				fi.writeReal(out)
				m.inits = append(m.inits, fi.name)
			} else {
				fi.writeReal(out)
			}
			if d.Body != nil && d.Name.IsExported() {
				fi.writeMock(out)
				fi.writeRecorder(out, recorder)
			}
			fmt.Fprintf(out, "\n")
		default:
			fmt.Fprintf(out, "--- Unknown Decl Type: %T\n", decl)
		}
	}

	fmt.Fprintf(out, "\n// Make sure gomock is used\n")
	fmt.Fprintf(out, "var _ = gomock.Any()\n")

	return nil
}

func loadInterfaceInfo(impPath string) (*ifInfo, error) {
	path, err := LookupImportPath(impPath)
	if err != nil {
		return nil, err
	}

	imports := make(map[string]string)
	ifInfo := newIfInfo("")

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
	pkgs, err := parser.ParseDir(fset, path, isGoFile, 0)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, i := range file.Imports {
				impPath := strings.Trim(i.Path.Value, "\"")
				if i.Name != nil {
					imports[i.Name.String()] = impPath
				} else {
					name, err := getPackageName(impPath, path)
					if err != nil {
						return nil, err
					}
					imports[name] = impPath
				}
			}

			for _, decl := range file.Decls {
				if d, ok := decl.(*ast.GenDecl); ok {
					if d.Tok == token.TYPE {
						for i := range d.Specs {
							t := d.Specs[i].(*ast.TypeSpec)
							ifInfo.addType(t, imports)
						}
					}
				}
			}
		}
	}

	return ifInfo, nil
}

func MockInterfaces(tmpPath, pkgName string) error {
	i := make(Interfaces)

	dst := filepath.Join(tmpPath, "src", pkgName, "_mocks_")
	err := os.MkdirAll(dst, 0700)
	if err != nil {
		return err
	}

	path, err := LookupImportPath(pkgName)
	if err != nil {
		return err
	}

	name, err := getPackageName(pkgName, path)
	if err != nil {
		return err
	}

	info, err := loadInterfaceInfo(pkgName)
	if err != nil {
		return err
	}

	info.filename = filepath.Join(dst, "ifmocks.go")

	i[name + "_mocks"] = info
	extPkg := markImport(pkgName, testMark)

	if err := i.genExtInterface(name + "_mocks", extPkg); err != nil {
		return err
	}

	// TODO: currently we need to use goimports to add missing imports, we
	// need to sort out our own imports, then we can switch to gofmt.
	if err := fixup(info.filename); err != nil {
		return err
	}

	return nil
}
