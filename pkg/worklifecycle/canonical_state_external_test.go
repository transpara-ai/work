package worklifecycle_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/transpara-ai/work/pkg/worklifecycle"
)

func TestTC3_CanonicalWorkStateExternalImmutability(t *testing.T) {
	typ := reflect.TypeOf(worklifecycle.CanonicalWorkState{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.IsExported() {
			t.Fatalf("CanonicalWorkState field %s is exported/writable", field.Name)
		}
	}

	ptrType := reflect.TypeOf(&worklifecycle.CanonicalWorkState{})
	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		if strings.HasPrefix(method.Name, "Set") {
			t.Fatalf("CanonicalWorkState exposes setter %s", method.Name)
		}
	}
}

func TestTC9_WorkLifecycleStdlibOnlyIncludesCanonicalDTO(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse package imports: %v", err)
	}
	for _, pkg := range pkgs {
		for fileName, file := range pkg.Files {
			if strings.HasSuffix(fileName, "_test.go") {
				continue
			}
			for _, spec := range file.Imports {
				path := strings.Trim(spec.Path.Value, `"`)
				if !strings.Contains(path, ".") {
					continue
				}
				t.Fatalf("%s imports non-stdlib package %q", fileName, path)
			}
			ast.Inspect(file, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if ok && sel.Sel.Name == "TaskStatus" {
					t.Fatalf("%s references TaskStatus; root work owns the adapter", fileName)
				}
				return true
			})
		}
	}
}
