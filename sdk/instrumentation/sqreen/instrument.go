// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
)

type instrumentationVisitor struct {
	// Instrumentation statistics of the currently instrumented package.
	stats instrumentationStats
	// Package path being instrumented. Used to generate unique hook names
	// prefixed by the package path.
	pkgPath string
	// The atomic load function must be added once
	atomicLoadDeclAdded bool
	// List of hookpoints in the current file being instrumented.
	instrumented []*hookpoint
	// List of files that were instrumented.
	instrumentedFiles []*dst.File
}

type instrumentationStats struct {
	ignored      []string
	instrumented []string
}

func (s *instrumentationStats) addIgnored(funcDecl *dst.FuncDecl) {
	s.ignored = append(s.ignored, funcDecl.Name.Name)
}

func (s *instrumentationStats) addInstrumented(funcDecl *dst.FuncDecl) {
	s.instrumented = append(s.instrumented, funcDecl.Name.Name)
}

func newInstrumentationVisitor(pkgPath string) *instrumentationVisitor {
	return &instrumentationVisitor{
		pkgPath: pkgPath,
	}
}

func (v *instrumentationVisitor) instrumentFuncDeclPre(funcDecl *dst.FuncDecl) {
	if shouldIgnoreFuncDecl(funcDecl) {
		v.stats.addIgnored(funcDecl)
		return
	}

	hook := newHookpoint(v.pkgPath, funcDecl)
	v.instrumented = append(v.instrumented, hook)

	funcDecl.Body.List = append([]dst.Stmt{hook.instrumentationStmt}, funcDecl.Body.List...)
}

func (v *instrumentationVisitor) instrument(root *dst.Package) []*dst.File {
	dstutil.Apply(root, v.instrumentPre, v.instrumentPost)
	return v.instrumentedFiles
}

func (v *instrumentationVisitor) instrumentPre(cursor *dstutil.Cursor) bool {
	switch node := cursor.Node().(type) {
	case *dst.FuncDecl:
		v.instrumentFuncDeclPre(node)
		// Note that we don't add the file metadata here in order to avoid to
		// infinite traversal because of adding new AST nodes while visiting it.

		// No need to go deeper than function declarations
		return false
	}
	return true
}

func (v *instrumentationVisitor) instrumentPost(cursor *dstutil.Cursor) bool {
	switch node := cursor.Node().(type) {
	case *dst.File:
		v.instrumentFilePost(node)
		v.instrumented = nil
	}
	return true
}

func (v *instrumentationVisitor) instrumentFilePost(file *dst.File) {
	if len(v.instrumented) == 0 {
		// Nothing got instrumented
		return
	}
	v.instrumentedFiles = append(v.instrumentedFiles, file)
	v.addFileMetadata(file)
}

func (v *instrumentationVisitor) addFileMetadata(file *dst.File) {
	addSqreenUnsafePackageImport(file)
	if !v.atomicLoadDeclAdded {
		v.addAtomicLoadFuncDecl(file)
	}
	for _, h := range v.instrumented {
		v.addHookMetadata(file, h)
	}
}

func (v *instrumentationVisitor) addHookDescriptorFuncDecl(file *dst.File, h *hookpoint) {
	file.Decls = append(file.Decls, h.descriptorFuncDecl)
}

func (v *instrumentationVisitor) addHookPrologLoadFuncDecl(file *dst.File, h *hookpoint) {
	file.Decls = append(file.Decls, h.prologLoadFuncDecl)
}

func (v *instrumentationVisitor) addHookMetadata(file *dst.File, h *hookpoint) {
	v.addHookPrologVarDecl(file, h)
	v.addHookPrologLoadFuncDecl(file, h)
	v.addHookDescriptorFuncDecl(file, h)
}

func (v *instrumentationVisitor) addHookPrologVarDecl(file *dst.File, h *hookpoint) {
	file.Decls = append(file.Decls, h.prologVarDecl)
}

func (v *instrumentationVisitor) addAtomicLoadFuncDecl(file *dst.File) {
	v.atomicLoadDeclAdded = true
	file.Decls = append(file.Decls, newLinkTimeSqreenAtomicLoadPointerFuncDecl())
}
