// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"fmt"
	"go/token"
	"log"
	"regexp"

	"github.com/dave/dst"
)

const sqreenAtomicLoadPointerFuncIdent = `_sqreen_atomic_load_pointer`

type hookpoint struct {
	id                  string
	descriptorFuncDecl  *dst.FuncDecl
	prologVarDecl       *dst.GenDecl
	prologLoadFuncDecl  *dst.FuncDecl
	instrumentationStmt dst.Stmt
}

func newHookpoint(pkgPath string, funcDecl *dst.FuncDecl) *hookpoint {
	id := normalizedHookpointID(pkgPath, funcDecl)

	epilogFuncType, epilogCallArgs := newSqreenEpilogFuncType(funcDecl.Type)
	prologFuncType, prologCallArgs := newSqreenPrologFuncType(funcDecl, epilogFuncType)

	prologVarIdent := fmt.Sprintf("_sqreen_hook_prolog_var_%s", id)
	prologVarDecl, prologValueSpec := newPrologVarDecl(prologVarIdent, prologFuncType)

	prologLoadFuncIdent := fmt.Sprintf("_sqreen_hook_prolog_load_%s", id)
	prologLoadFuncDecl := newPrologLoadFuncDecl(prologLoadFuncIdent, prologValueSpec)

	descriptorFuncIdent := fmt.Sprintf("_sqreen_hook_descriptor_%s", id)
	descriptorFuncDecl := newHookDescriptorFuncDecl(descriptorFuncIdent, funcDecl)

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
										&dst.CallExpr{
											Fun: newSqreenUnsafePointerType(),
											Args: []dst.Expr{
												&dst.UnaryExpr{
													Op: token.AND,
													X:  dst.NewIdent(prologVarName),
												},
											},
										},
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
	if funcDecl.Recv != nil {
		callbackTypeParamList.List[0].Type = newEmptyInterfaceType()
	}
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
func newHookDescriptorFuncDecl(ident string, funcDecl *dst.FuncDecl) *dst.FuncDecl {
	return &dst.FuncDecl{
		Name: dst.NewIdent(ident),
		Type: &dst.FuncType{
			Params: &dst.FieldList{},
			Results: &dst.FieldList{
				List: []*dst.Field{
					{
						Type: &dst.InterfaceType{Methods: &dst.FieldList{Opening: true, Closing: true}},
					},
				},
			},
		},
		Body: &dst.BlockStmt{
			List: []dst.Stmt{
				&dst.ReturnStmt{
					Results: []dst.Expr{
						newFunctionAddressExpr(funcDecl),
					},
				},
			},
		},
		Decs: dst.FuncDeclDecorations{
			NodeDecs: dst.NodeDecs{
				Before: dst.NewLine,
				Start: dst.Decorations{
					fmt.Sprintf("//go:linkname %[1]s %[1]s\n", ident),
				},
			},
		},
	}
}

// Return link time function declaration for the atomic load pointer function.
func newLinkTimeSqreenAtomicLoadPointerFuncDecl() *dst.FuncDecl {
	ftype := &dst.FuncType{
		Params: &dst.FieldList{
			List: []*dst.Field{{Type: newSqreenUnsafePointerType()}},
		},
		Results: &dst.FieldList{
			List: []*dst.Field{{Type: newSqreenUnsafePointerType()}},
		},
	}
	return newLinkTimeForwardFuncDecl(sqreenAtomicLoadPointerFuncIdent, ftype)
}
