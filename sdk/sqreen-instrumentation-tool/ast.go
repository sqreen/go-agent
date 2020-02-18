// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"fmt"
	"go/token"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dave/dst"
)

const (
	sqreenAtomicLoadPointerFuncIdent = `_sqreen_atomic_load_pointer`

	sqreenHookDescriptorIdentPrefix                  = `_sqreen_hook_descriptor_`
	sqreenHookDescriptorTypeIdent                    = sqreenHookDescriptorIdentPrefix + `type`
	sqreenHookDescriptorFuncIdentFormat              = sqreenHookDescriptorIdentPrefix + `%s`
	sqreenHookDescriptorFuncIdentPrefixOfMainPackage = sqreenHookDescriptorIdentPrefix + `main_`

	sqreenPrologVarIdentPrefix = `_sqreen_hook_prolog_var_`
	sqreenPrologVarIdentFormat = sqreenPrologVarIdentPrefix + `%s`

	sqreenPrologLoadFuncIdentFormat = `_sqreen_hook_prolog_load_%s`
)

// The hookpoint structure holds every AST node required during the
// instrumentation of a file.
type hookpoint struct {
	id                  string
	descriptorFuncDecl  *dst.FuncDecl
	prologVarDecl       *dst.GenDecl
	prologLoadFuncDecl  *dst.FuncDecl
	instrumentationStmt dst.Stmt
}

func newHookpoint(pkgPath string, funcDecl *dst.FuncDecl, descriptorTypeIdent string, descriptorValueInitializer hookDescriptorValueInitializer) *hookpoint {
	id := normalizedHookpointID(pkgPath, funcDecl)

	epilogFuncType, epilogCallArgs := newSqreenEpilogFuncType(funcDecl.Type)
	prologFuncType, prologCallArgs := newSqreenPrologFuncType(funcDecl, epilogFuncType)

	prologVarIdent := fmt.Sprintf(sqreenPrologVarIdentFormat, id)
	prologVarDecl, prologValueSpec := newPrologVarDecl(prologVarIdent, prologFuncType)

	prologLoadFuncIdent := fmt.Sprintf(sqreenPrologLoadFuncIdentFormat, id)
	prologLoadFuncDecl := newPrologLoadFuncDecl(prologLoadFuncIdent, prologValueSpec)

	descriptorFuncIdent := fmt.Sprintf(sqreenHookDescriptorFuncIdentFormat, id)
	descriptorFuncDecl := newHookDescriptorFuncDecl(descriptorFuncIdent, funcDecl, prologVarIdent, descriptorValueInitializer)

	instrumentationStmt := newInstrumentationStmt(prologLoadFuncIdent, prologCallArgs, epilogCallArgs)

	return &hookpoint{
		id:                  id,
		prologLoadFuncDecl:  prologLoadFuncDecl,
		descriptorFuncDecl:  descriptorFuncDecl,
		prologVarDecl:       prologVarDecl,
		instrumentationStmt: instrumentationStmt,
	}
}

func normalizedHookpointID(pkgPath string, node *dst.FuncDecl) string {
	var receiver string
	if node.Recv != nil {
		t := node.Recv.List[0].Type
	loop:
		for {
			switch actual := t.(type) {
			default:
				log.Fatalf("unexpected type %T\n", actual)

			case *dst.StarExpr:
				t = actual.X

			case *dst.Ident:
				receiver = actual.Name
				break loop
			}
		}
		receiver += "_"
	}
	pkgPath = normalizedPkgPath(pkgPath)
	return fmt.Sprintf("%s_%s%s", pkgPath, receiver, node.Name)
}

func normalizedPkgPath(pkgPath string) string {
	return regexp.MustCompile(`[/.\-@]`).ReplaceAllString(pkgPath, "_")
}

// Return the global prolog variable declaration.
func newPrologVarDecl(ident string, typ dst.Expr) (*dst.GenDecl, *dst.ValueSpec) {
	typ = &dst.StarExpr{X: dst.Clone(typ).(dst.Expr)}
	return newVarDecl(ident, typ)
}

// Return the function declaration loading the global prolog variable using
// Sqreen's atomic load pointer function.
func newPrologLoadFuncDecl(ident string, prologVarSpec *dst.ValueSpec) *dst.FuncDecl {
	prologVarName := prologVarSpec.Names[0].Name
	prologVarType := prologVarSpec.Type
	retType := dst.Clone(prologVarType).(dst.Expr)
	retCastType := dst.Clone(prologVarType).(dst.Expr)

	return &dst.FuncDecl{
		Name: dst.NewIdent(ident),
		Type: &dst.FuncType{
			Params: &dst.FieldList{},
			Results: &dst.FieldList{
				List: []*dst.Field{
					{
						Type: retType,
					},
				},
			},
		},
		Body: &dst.BlockStmt{
			List: []dst.Stmt{
				&dst.ReturnStmt{
					Results: []dst.Expr{
						&dst.CallExpr{
							Fun: retCastType,
							Args: []dst.Expr{
								&dst.CallExpr{
									Fun: dst.NewIdent(sqreenAtomicLoadPointerFuncIdent),
									Args: []dst.Expr{
										newCastValueExpr(
											newPointerTypeOf(newSqreenUnsafePointerType()),
											newCastValueExpr(newSqreenUnsafePointerType(), newIdentAddressExpr(dst.NewIdent(prologVarName)))),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Return the instrumentation statement node to be added to a function body.
func newInstrumentationStmt(prologLoadFuncIdent string, prologCallArgs, epilogCallArgs []dst.Expr) dst.Stmt {
	const sqreenPrologVarIdent = "_sqreen_prolog"
	const sqreenPrologAbortErrorVarIdent = "_sqreen_prolog_abort_err"
	const sqreenEpilogVarIdent = "_sqreen_epilog"
	const nilIdent = "nil"

	return &dst.BlockStmt{
		List: []dst.Stmt{
			// if _sqreen_prolog := <prologLoadFuncIdent>(); _sqreen_prolog != nil { ... }
			&dst.IfStmt{
				Init: &dst.AssignStmt{
					Lhs: []dst.Expr{dst.NewIdent(sqreenPrologVarIdent)},
					Tok: token.DEFINE,
					Rhs: []dst.Expr{&dst.CallExpr{Args: []dst.Expr{}, Fun: dst.NewIdent(prologLoadFuncIdent)}},
				},
				Cond: &dst.BinaryExpr{
					X:  dst.NewIdent(sqreenPrologVarIdent),
					Op: token.NEQ,
					Y:  dst.NewIdent(nilIdent),
				},
				Body: &dst.BlockStmt{
					List: []dst.Stmt{
						// _sqreen_epilog, _sqreen_prolog_abort_err := (*sqreen_prolog)(<args>)
						&dst.AssignStmt{
							Lhs: []dst.Expr{
								dst.NewIdent(sqreenEpilogVarIdent),
								dst.NewIdent(sqreenPrologAbortErrorVarIdent),
							},
							Tok: token.DEFINE,
							Rhs: []dst.Expr{
								&dst.CallExpr{
									Fun:      &dst.StarExpr{X: dst.NewIdent(sqreenPrologVarIdent)},
									Args:     prologCallArgs,
									Ellipsis: false,
									Decs:     dst.CallExprDecorations{},
								},
							},
						},
						// if _sqreen_epilog != nil { defer _sqreen_epilog(<args>) }
						&dst.IfStmt{
							Cond: &dst.BinaryExpr{
								X:  dst.NewIdent(sqreenEpilogVarIdent),
								Op: token.NEQ,
								Y:  dst.NewIdent(nilIdent),
							},
							Body: &dst.BlockStmt{
								List: []dst.Stmt{
									&dst.DeferStmt{
										Call: &dst.CallExpr{
											Fun:  dst.NewIdent(sqreenEpilogVarIdent),
											Args: epilogCallArgs,
										},
									},
								},
							},
						},
						// if _sqreen_prolog_abort_err != nil { return }
						&dst.IfStmt{
							Cond: &dst.BinaryExpr{
								X:  dst.NewIdent(sqreenPrologAbortErrorVarIdent),
								Op: token.NEQ,
								Y:  dst.NewIdent(nilIdent),
							},
							Body: &dst.BlockStmt{
								List: []dst.Stmt{
									&dst.ReturnStmt{},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Return the epilog type of the given function type.
// `f(<params>) <results>` returns `func(<*params>) (<epilog type>, error)`
func newSqreenPrologFuncType(funcDecl *dst.FuncDecl, epilogType *dst.FuncType) (prologType *dst.FuncType, callParams []dst.Expr) {
	funcType := funcDecl.Type
	callbackTypeParamList, callbackCallParams := newSqreenCallbackParams(funcDecl.Recv, funcType.Params, "_sqreen_param")
	return &dst.FuncType{
		Params: callbackTypeParamList,
		Results: &dst.FieldList{
			List: []*dst.Field{
				{
					Type: epilogType,
				},
				{
					Type: dst.NewIdent("error"),
				},
			},
		},
	}, callbackCallParams
}

// Return the epilog type of the given function type.
// `f(<params>) <results>` returns `func(<*results>)`
func newSqreenEpilogFuncType(funcType *dst.FuncType) (epilogType *dst.FuncType, callParams []dst.Expr) {
	callbackTypeParamList, callbackCallParams := newSqreenCallbackParams(nil, funcType.Results, "_sqreen_result")
	return &dst.FuncType{
		Params:  callbackTypeParamList,
		Results: &dst.FieldList{},
	}, callbackCallParams
}

// newSqreenCallback walks the given function parameters and returns the
// parameter for the callback (prolog or epilog), along with the list of call
// arguments.
func newSqreenCallbackParams(recv *dst.FieldList, params *dst.FieldList, ignoredParamPrefix string) (callbackTypeParamList *dst.FieldList, callbackCallParams []dst.Expr) {
	var callbackTypeParams []*dst.Field
	var hookedParams []*dst.Field
	if recv != nil {
		hookedParams = recv.List
	}
	if params != nil {
		hookedParams = append(hookedParams, params.List...)
	}
	p := 0
	for _, hookedParam := range hookedParams {
		callbackTypeParam := &dst.Field{Type: newSqreenCallbackParamType(hookedParam.Type)}
		if len(hookedParam.Names) == 0 {
			// Case where the parameter has no name such as f(string): no longer
			// ignore it and name it.
			// - The hooked function parameter must be named.
			hookedParam.Names = []*dst.Ident{newSqreenParamIdent(ignoredParamPrefix, p)}
			// - The callback type expects this parameter type.
			callbackTypeParams = append(callbackTypeParams, callbackTypeParam)
			// - The callback call must pass the hooked function parameter.
			callbackCallParams = append(callbackCallParams, newSqreenCallbackCallParam(newSqreenParamIdent(ignoredParamPrefix, p)))
			p++
		} else {
			// Case where the parameters are named, but still possibly ignored.
			for _, name := range hookedParam.Names {
				if name.Name == "_" {
					// Case where the parameter is ignored using `_` such as
					// f(_ string):  no longer ignore it and name it.
					*name = *newSqreenParamIdent(ignoredParamPrefix, p)
				}
				callbackTypeParam = dst.Clone(callbackTypeParam).(*dst.Field)

				// The callback type expects this parameter type.
				callbackTypeParams = append(callbackTypeParams, callbackTypeParam)
				// The callback call must pass the hooked function parameter.
				callbackCallParams = append(callbackCallParams, newSqreenCallbackCallParam(dst.NewIdent(name.Name)))
				p++
			}
		}
	}
	return &dst.FieldList{List: callbackTypeParams}, callbackCallParams
}

func newSqreenCallbackParamType(hookedParamType dst.Expr) dst.Expr {
	typ := dst.Clone(hookedParamType).(dst.Expr)
	if variadic, ok := typ.(*dst.Ellipsis); ok {
		typ = &dst.ArrayType{Elt: variadic.Elt}
	}
	return &dst.StarExpr{X: typ}
}

func newSqreenCallbackCallParam(ident *dst.Ident) dst.Expr {
	return newIdentAddressExpr(ident)
}

func newSqreenParamIdent(prefix string, p int) *dst.Ident {
	return dst.NewIdent(fmt.Sprintf("%s%d", prefix, p))
}

// Return the hook descriptor function declaration which returns the hook
// descriptor structure.
func newHookDescriptorFuncDecl(ident string, funcDecl *dst.FuncDecl, prologVarIdent string, newDescriptorValueInitializer hookDescriptorValueInitializer) *dst.FuncDecl {
	const descriptorParamName = `_sqreen_hd`
	return &dst.FuncDecl{
		Decs: dst.FuncDeclDecorations{
			NodeDecs: dst.NodeDecs{
				Before: dst.NewLine,
				Start: dst.Decorations{
					"//go:nosplit", // save some instructions
					fmt.Sprintf("//go:linkname %[1]s %[1]s\n", ident),
				},
			},
		},
		Name: dst.NewIdent(ident),
		Type: &dst.FuncType{
			Params: &dst.FieldList{
				List: []*dst.Field{
					{
						Names: []*dst.Ident{dst.NewIdent(descriptorParamName)},
						Type:  &dst.StarExpr{X: dst.NewIdent(sqreenHookDescriptorTypeIdent)},
					},
				},
			},
			Results: &dst.FieldList{},
		},
		Body: &dst.BlockStmt{
			List: []dst.Stmt{
				&dst.AssignStmt{
					Lhs: []dst.Expr{
						&dst.StarExpr{X: dst.NewIdent(descriptorParamName)},
					},
					Tok: token.ASSIGN,
					Rhs: []dst.Expr{
						newDescriptorValueInitializer(newFunctionValueExpr(funcDecl), newIdentAddressExpr(dst.NewIdent(prologVarIdent))),
					},
				},
			},
		},
	}
}

// Return link time function declaration for the atomic load pointer function.
func newLinkTimeSqreenAtomicLoadPointerFuncDecl() *dst.FuncDecl {
	ftype := &dst.FuncType{
		Params: &dst.FieldList{
			List: []*dst.Field{{Type: &dst.StarExpr{X: newSqreenUnsafePointerType()}}},
		},
		Results: &dst.FieldList{
			List: []*dst.Field{{Type: newSqreenUnsafePointerType()}},
		},
	}
	return newLinkTimeForwardFuncDecl(sqreenAtomicLoadPointerFuncIdent, ftype)
}

type hookDescriptorValueInitializer func(Func, Prolog dst.Expr) dst.Expr

// Return the type declaration for
// ```
// type _sqreen_hook_descriptor_type = struct {
//   Func, Prolog interface{}
// }
// ```
func newHookDescriptorType() (*dst.GenDecl, *dst.TypeSpec, hookDescriptorValueInitializer) {
	spec := &dst.TypeSpec{
		Name:   dst.NewIdent(sqreenHookDescriptorTypeIdent),
		Assign: true,
		Type: &dst.StructType{
			Fields: &dst.FieldList{
				List: []*dst.Field{
					{
						Names: []*dst.Ident{
							dst.NewIdent("Func"),
							dst.NewIdent("Prolog"),
						},
						Type: newEmptyInterfaceType(),
					},
				},
			},
		},
	}

	typ := &dst.GenDecl{
		Tok: token.TYPE,
		Specs: []dst.Spec{
			spec,
		},
	}

	valInitializer := func(Func, Prolog dst.Expr) dst.Expr {
		return &dst.CompositeLit{
			Type: dst.NewIdent(sqreenHookDescriptorTypeIdent),
			Elts: []dst.Expr{
				&dst.KeyValueExpr{
					Key:   dst.NewIdent("Func"),
					Value: Func,
				},
				&dst.KeyValueExpr{
					Key:   dst.NewIdent("Prolog"),
					Value: Prolog,
				},
			},
		}
	}

	return typ, spec, valInitializer
}

func isFileNameIgnored(file string) bool {
	filename := filepath.Base(file)
	// Don't instrument cgo files
	if strings.Contains(filename, "cgo") {
		return true
	}
	// Don't instrument the go module table file.
	if filename == "_gomod_.go" {
		return true
	}
	return false
}

var ignoredPkgPrefixes = []string{
	"runtime",
	"sync",
	"reflect",
	"internal",
	"unsafe",
	"syscall",
	"time",
	"math",
}

var limitedInstrumentationPkgPrefixes = []string{
	"github.com/sqreen/go-agent/internal/protection",
	"database/sql",
}

func isPackageNameIgnored(pkg string, fullInstrumentation bool) bool {
	for _, prefix := range ignoredPkgPrefixes {
		if strings.HasPrefix(pkg, prefix) {
			return true
		}
	}

	if fullInstrumentation {
		return false
	}

	// Non-full instrumentation mode, limited to a set of given package
	for _, prefix := range limitedInstrumentationPkgPrefixes {
		if strings.HasPrefix(pkg, prefix) {
			return false
		}
	}

	return true
}

func shouldIgnoreFuncDecl(funcDecl *dst.FuncDecl) bool {
	fname := funcDecl.Name.Name
	// don't instrument:
	// - `_`: explicitly ignored function names.
	// - `init`: package init functions.
	// - `.*noescape.*`: any function name containing `noescape` since we would
	//    likely break it.
	// - functions having //go:nosplit directives because they are usually low-level
	//   functions.
	// - functions having //sqreen:ignore directives.
	return funcDecl.Body == nil ||
		fname == "_" ||
		fname == "init" ||
		strings.Contains(fname, "noescape") ||
		hasSqreenIgnoreDirective(funcDecl) ||
		hasGoNoSplitDirective(funcDecl)
}

func isHookDescriptorFuncInMainPackage(ident string) bool {
	return strings.HasPrefix(ident, sqreenHookDescriptorFuncIdentPrefixOfMainPackage)
}
