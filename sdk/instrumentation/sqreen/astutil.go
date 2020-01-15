// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"fmt"
	"go/printer"
	"go/token"
	"io"
	"strconv"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

const (
	sqreenUnsafePackageName = `_sqreen_unsafe_`
	sqreenIgnoreDirective   = `//sqreen:ignore`
)

func newVarDecl(ident string, typ dst.Expr) (*dst.GenDecl, *dst.ValueSpec) {
	varSpec := &dst.ValueSpec{
		Names: []*dst.Ident{
			dst.NewIdent(ident),
		},
		Type: typ,
	}
	return &dst.GenDecl{
		Tok: token.VAR,
		Specs: []dst.Spec{
			varSpec,
		},
	}, varSpec
}

func newLinkTimeForwardFuncDecl(ident string, ftype *dst.FuncType) *dst.FuncDecl {
	return &dst.FuncDecl{
		Decs: dst.FuncDeclDecorations{
			NodeDecs: dst.NodeDecs{
				Before: dst.NewLine,
				Start: dst.Decorations{
					fmt.Sprintf("//go:linkname %[1]s %[1]s\n", ident),
				},
			},
		},
		Name: dst.NewIdent(ident),
		Type: ftype,
	}
}

// Return expression for type `_sqreen_unsafe_.Pointer`
func newSqreenUnsafePointerType() dst.Expr {
	return newQualifiedIdent(sqreenUnsafePackageName, "Pointer")
}

// Return qualified identifier for `pkgName.ident`
func newQualifiedIdent(pkgName, ident string) dst.Expr {
	return &dst.SelectorExpr{X: dst.NewIdent(pkgName), Sel: dst.NewIdent(ident)}
}

// addImport adds an import to the file
func addNamedImport(file *dst.File, name string, path string) {
	imp := &dst.ImportSpec{
		Name: dst.NewIdent(name),
		Path: &dst.BasicLit{Kind: token.STRING, Value: strconv.Quote(path)},
	}
	file.Imports = append(file.Imports, imp)
	file.Decls = append([]dst.Decl{&dst.GenDecl{
		Tok:   token.IMPORT,
		Specs: []dst.Spec{imp},
	}}, file.Decls...)
}

// Return expression for `&ident`
func newIdentAddressExpr(ident *dst.Ident) dst.Expr {
	return &dst.UnaryExpr{Op: token.AND, X: ident}
}

// Return expression for `interface{}`
func newEmptyInterfaceType() dst.Expr {
	return &dst.InterfaceType{Methods: &dst.FieldList{Opening: true, Closing: true}}
}

// Return expression for `expr.sel`
func newSelectorExpr(expr dst.Expr, sel string) *dst.SelectorExpr {
	return &dst.SelectorExpr{
		X:   expr,
		Sel: dst.NewIdent(sel),
	}
}

// Return method address expression `(<receiver type>).<method name>` for
// the given function declaration.
func newMethodAddressExpr(fn *dst.FuncDecl) dst.Expr {
	t := fn.Recv.List[0].Type
	return newSelectorExpr(&dst.ParenExpr{X: t}, fn.Name.Name)
}

// Return the address expression for the given function declaration.
// It can be either a method address or a function address.
func newFunctionAddressExpr(fn *dst.FuncDecl) (addr dst.Expr) {
	if fn.Recv == nil {
		addr = fn.Name
	} else {
		addr = newMethodAddressExpr(fn)
	}
	return dst.Clone(addr).(dst.Expr)
}

func hasSqreenIgnoreDirective(node dst.Node) bool {
	for _, c := range node.Decorations().Start.All() {
		if c == sqreenIgnoreDirective {
			return true
		}
	}
	return false
}

func addSqreenUnsafePackageImport(file *dst.File) {
	addNamedImport(file, sqreenUnsafePackageName, "unsafe")
}

func writeFile(file *dst.File, w io.Writer) error {
	fset, af, err := decorator.RestoreFile(file)
	if err != nil {
		return err
	}
	return printer.Fprint(w, fset, af)
}

func hasGoNoSplitDirective(funcDecl *dst.FuncDecl) bool {
	const pragma = `//go:nosplit`
	for _, c := range funcDecl.Decs.Start.All() {
		if c == pragma {
			return true
		}
	}
	return false
}
