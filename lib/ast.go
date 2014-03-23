// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"go/ast"
	"reflect"
)

// breakLoops sets all *ast.Object instances to nil to try and stop loops that
// break gob.
func breakLoops(expr ast.Expr) {
	doBreakLoops(reflect.ValueOf(expr))
}

func doBreakLoops(v reflect.Value) {
	if !v.IsValid() {
		return
	}

	objNil := (*ast.Object)(nil)
	if v.Type() == reflect.TypeOf(objNil) {
		v.Set(reflect.ValueOf(objNil))
		return
	}

	switch v.Kind() {
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			doBreakLoops(v.Index(i))
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			doBreakLoops(v.Index(i))
		}
	case reflect.Interface:
		doBreakLoops(v.Elem())
	case reflect.Ptr:
		doBreakLoops(v.Elem())
	case reflect.Struct:
		for i, n := 0, v.NumField(); i < n; i++ {
			doBreakLoops(v.Field(i))
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			doBreakLoops(v.MapIndex(k))
		}
	}
}
