// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"bytes"
	"fmt"
	"io"
	"sort"

	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
	"github.com/sqreen/go-agent/internal"
)

type instrumentationVisitor struct {
	// Instrumentation statistics of the currently instrumented package.
	stats instrumentationStats
	// Package path being instrumented. Used to generate unique hook names
	// prefixed by the package path.
	pkgPath string
	// False when the first file is being instrumented in order to add
	// metadata that must appear once.
	fileMetadataOnce bool
	// List of hookpoints in the current file being instrumented.
	instrumented []*hookpoint
	// Map of instrumented files along with there hookpoints
	instrumentedFiles map[*dst.File][]*hookpoint
	// Hook descriptor type declaration node. It will be added to the file
	// metadata.
	hookDescriptorTypeIdent string
	// The hook descriptor value initializer used by the hook descriptor function
	// in order to create a new descriptor value.
	newHookDescriptorValueInitializer hookDescriptorValueInitializer
	// The hook descriptor type declaration added once per instrumented package
	// and used by hook descriptor functions to return a value of that type.
	hookDescriptorTypeDecl *dst.GenDecl
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
	hookDescriptorTypeDecl, hookDescriptorTypeSpec, newDescriptorValueInitializer := newHookDescriptorType()
	hookDescriptorTypeIdent := hookDescriptorTypeSpec.Name.Name
	return &instrumentationVisitor{
		pkgPath:                           pkgPath,
		instrumentedFiles:                 make(map[*dst.File][]*hookpoint),
		hookDescriptorTypeIdent:           hookDescriptorTypeIdent,
		hookDescriptorTypeDecl:            hookDescriptorTypeDecl,
		newHookDescriptorValueInitializer: newDescriptorValueInitializer,
	}
}

func (v *instrumentationVisitor) instrumentFuncDeclPre(funcDecl *dst.FuncDecl) {
	if shouldIgnoreFuncDecl(funcDecl) {
		v.stats.addIgnored(funcDecl)
		return
	}

	hook := newHookpoint(v.pkgPath, funcDecl, v.hookDescriptorTypeIdent, v.newHookDescriptorValueInitializer)
	v.instrumented = append(v.instrumented, hook)

	funcDecl.Body.List = append([]dst.Stmt{hook.instrumentationStmt}, funcDecl.Body.List...)
}

func (v *instrumentationVisitor) instrument(root *dst.Package) (instrumented map[*dst.File][]*hookpoint) {
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
	}
	return true
}

func (v *instrumentationVisitor) instrumentFilePost(file *dst.File) {
	if len(v.instrumented) == 0 {
		// Nothing got instrumented
		return
	}
	// Add file-level instrumentation metadata
	v.addFileMetadata(file, v.instrumented)
	// Save the list of hooks
	v.instrumentedFiles[file] = v.instrumented
	v.instrumented = nil
}

func (v *instrumentationVisitor) addFileMetadata(file *dst.File, instrumented []*hookpoint) {
	addSqreenUnsafePackageImport(file)
	if !v.fileMetadataOnce {
		v.fileMetadataOnce = true
		v.addAtomicLoadFuncDecl(file)
		v.addHookDescriptorType(file)
	}
	for _, h := range instrumented {
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
	file.Decls = append(file.Decls, newLinkTimeSqreenAtomicLoadPointerFuncDecl())
}

func (v *instrumentationVisitor) addHookDescriptorType(file *dst.File) {
	file.Decls = append(file.Decls, v.hookDescriptorTypeDecl)
}

// Write into `w` the Go sources of the hook table for the list of hook
// descriptor function `hooks`.
func writeHookTable(w io.Writer, hooks []string) error {
	sort.Strings(hooks)

	// In case the hook descriptor type hasn't been created, we recreate the
	// type alias again in the hook table file and with a distinct name.
	const (
		tableFormat = `var _sqreen_hook_table_array = [...]func(*_sqreen_hook_table_hook_descriptor_type){%s
}

type _sqreen_hook_table_type = []func(*_sqreen_hook_table_hook_descriptor_type)
type _sqreen_instrumentation_descriptor_type = struct {
	Version   string
	HookTable _sqreen_hook_table_type
}

//go:linkname _sqreen_instrumentation_descriptor _sqreen_instrumentation_descriptor
var _sqreen_instrumentation_descriptor = &_sqreen_instrumentation_descriptor_type{
	Version: %q,
	HookTable: _sqreen_hook_table_array[:],
}
`
		tableInitListEntryFormat = "\n\t%s,"

		hookDescriptorForwardFuncDeclFormat = `//go:linkname %[1]s %[1]s
func %[1]s(*_sqreen_hook_table_hook_descriptor_type)

`
		fileFormat = `package main

import _ "unsafe"

type _sqreen_hook_table_hook_descriptor_type = struct {	Func, Prolog interface{} }

%s

%s
`
	)

	var tableInitList, hookDescriptorForwardFuncDecls bytes.Buffer

	for _, hookDescriptorFuncName := range hooks {
		// Create the hook table initializer entry line
		tableInitListEntry := fmt.Sprintf(tableInitListEntryFormat, hookDescriptorFuncName)
		if _, err := io.WriteString(&tableInitList, tableInitListEntry); err != nil {
			return err
		}

		// We are writing a file for the main package so we don't need to forward
		// declare the hook descriptor functions that are defined in the main
		// package.
		if isHookDescriptorFuncInMainPackage(hookDescriptorFuncName) {
			continue
		}

		// Create forward declaration of the hook descriptor function
		hookDescriptorForwardFuncDecl := fmt.Sprintf(hookDescriptorForwardFuncDeclFormat, hookDescriptorFuncName)
		if _, err := io.WriteString(&hookDescriptorForwardFuncDecls, hookDescriptorForwardFuncDecl); err != nil {
			return err
		}
	}

	hookTableVar := fmt.Sprintf(tableFormat, &tableInitList, internal.Version())
	_, err := io.WriteString(w, fmt.Sprintf(fileFormat, &hookDescriptorForwardFuncDecls, hookTableVar))
	return err
}
