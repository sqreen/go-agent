// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
)

type Instrumenter interface {
	IsIgnored() bool
	AddFile(src string) error
	Instrument() ([]*dst.File, error)
	WriteInstrumentedFiles(packageBuildDir string, instrumented []*dst.File) (srcdst map[string]string, err error)
	WriteExtraFiles() ([]string, error)
}

type packageInstrumentationHelper struct {
	parsedFiles       map[string]*dst.File
	parsedFileSources map[*dst.File]string
	fset              *token.FileSet
}

// AddFile parses the given Go source file `src` and adds it to the set of
// files to instrument if it is not ignored by a directive.
func (h *packageInstrumentationHelper) AddFile(src string) error {
	// Check if the instrumentation should be skipped for this filename
	if isFileNameIgnored(src) {
		log.Println("skipping instrumentation of file", src)
		return nil
	}

	log.Printf("parsing file `%s`", src)
	if h.fset != nil {
		// The token fileset is required to later create the package node.
		h.fset = token.NewFileSet()
	}
	file, err := decorator.ParseFile(h.fset, src, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// Check if there is a file-level ignore directive
	if hasSqreenIgnoreDirective(file) {
		log.Printf("file `%s` skipped due to ignore directive", src)
		return nil
	}
	if h.parsedFiles == nil {
		h.parsedFiles = make(map[string]*dst.File)
		h.parsedFileSources = make(map[*dst.File]string)
	}
	h.parsedFiles[src] = file
	h.parsedFileSources[file] = src
	return nil
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

func (h *packageInstrumentationHelper) instrument(v instrumentationVisitorFace) (instrumented []*dst.File, err error) {
	if len(h.parsedFiles) == 0 {
		log.Println("nothing to instrument")
		return nil, nil
	}

	root, err := dst.NewPackage(h.fset, h.parsedFiles, nil, nil)
	if err != nil {
		return nil, err
	}

	return v.instrument(root), nil
}

func (h *packageInstrumentationHelper) WriteInstrumentedFiles(buildDirPath string, instrumentedFiles []*dst.File) (srcdst map[string]string, err error) {
	srcdst = make(map[string]string, len(instrumentedFiles))
	for _, node := range instrumentedFiles {
		src := h.parsedFileSources[node]
		filename := filepath.Base(src)
		dest := filepath.Join(buildDirPath, filename)
		output, err := os.Create(dest)
		if err != nil {
			return nil, err
		}
		defer output.Close()
		// Add a go line directive in order to map it to its original source file.
		// Note that otherwise it uses the build directory but it is trimmed by the
		// compiler - so you end up with filenames without any leading path (eg.
		// myfile.go) leading to broken debuggers or stack traces.
		output.WriteString(fmt.Sprintf("//line %s:1\n", src))
		if err := writeFile(node, output); err != nil {
			return nil, err
		}
		srcdst[src] = dest
	}
	return srcdst, nil
}

type defaultPackageInstrumentation struct {
	packageInstrumentationHelper
	instrumentedFiles   map[*dst.File][]*hookpoint
	pkgPath             string
	fullInstrumentation bool
	hookListFilepath    string
	packageBuildDir     string
}

func newDefaultPackageInstrumentation(pkgPath string, fullInstrumentation bool, packageBuildDir string) *defaultPackageInstrumentation {
	projectBuildDir := path.Join(packageBuildDir, "..")
	hookListFilepath := getHookListFilepath(projectBuildDir)

	return &defaultPackageInstrumentation{
		pkgPath:             pkgPath,
		fullInstrumentation: fullInstrumentation,
		hookListFilepath:    hookListFilepath,
		packageBuildDir:     packageBuildDir,
	}
}

func (h *defaultPackageInstrumentation) IsIgnored() bool {
	// Check if the instrumentation should be skipped for this package name.
	if h.isPackageNameIgnored() {
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

// List of packages to instrument when not in full instrumentation mode.
var (
	// List of package path prefixes to instrument. A package is instrumented
	// iif its package path begins with one of the following prefixes.
	limitedInstrumentationPkgPathPrefixes = []string{
		"github.com/sqreen/go-agent/internal/protection",
		"database/sql",
	}

	// List of package paths to instrument. A package is instrumented iif it is
	// equal to one of the following package paths.
	limitedInstrumentationPkgPaths = []string{
		"os",
	}
)

func (h *defaultPackageInstrumentation) isPackageNameIgnored() bool {
	for _, prefix := range ignoredPkgPrefixes {
		if strings.HasPrefix(h.pkgPath, prefix) {
			return true
		}
	}

	if h.fullInstrumentation {
		return false
	}

	// Non-full instrumentation mode is limited to a set of packages
	for _, pkgPath := range limitedInstrumentationPkgPaths {
		if h.pkgPath == pkgPath {
			return false
		}
	}

	for _, prefix := range limitedInstrumentationPkgPathPrefixes {
		if strings.HasPrefix(h.pkgPath, prefix) {
			return false
		}
	}

	return true
}

func (h *defaultPackageInstrumentation) Instrument() (instrumented []*dst.File, err error) {
	h.instrumentedFiles = make(map[*dst.File][]*hookpoint)
	v := newDefaultPackageInstrumentationVisitor(h.pkgPath, h.instrumentedFiles)
	return h.packageInstrumentationHelper.instrument(v)
}

func (h *defaultPackageInstrumentation) writeHookList(hookList *os.File) (count int, err error) {
	for _, hooks := range h.instrumentedFiles {
		for _, hook := range hooks {
			if _, err = hookList.WriteString(fmt.Sprintf("%s\n", hook.descriptorFuncDecl.Name.Name)); err != nil {
				return count, err
			}
			count += 1
		}
	}
	return count, nil
}

func (h *defaultPackageInstrumentation) WriteExtraFiles() (extra []string, err error) {
	// Add the hook IDs to the hook list file.
	hookListFile, err := openHookListFile(h.hookListFilepath)
	if err != nil {
		return nil, err
	}
	defer hookListFile.Close()
	count, err := h.writeHookList(hookListFile)
	if err != nil {
		return nil, err
	}
	log.Printf("added %d hooks to the hook list\n", count)
	return nil, nil
}

type mainPackageInstrumentation struct {
	*defaultPackageInstrumentation
}

func newMainPackageInstrumentation(pkgPath string, fullInstrumentation bool, packageBuildDir string) *mainPackageInstrumentation {
	return &mainPackageInstrumentation{
		defaultPackageInstrumentation: newDefaultPackageInstrumentation(pkgPath, fullInstrumentation, packageBuildDir),
	}
}

func (m *mainPackageInstrumentation) IsIgnored() bool {
	return false
}

func (m *mainPackageInstrumentation) Instrument() ([]*dst.File, error) {
	if m.defaultPackageInstrumentation.IsIgnored() {
		return nil, nil
	}
	return m.defaultPackageInstrumentation.Instrument()
}

func (m *mainPackageInstrumentation) WriteExtraFiles() (extra []string, err error) {
	if !m.defaultPackageInstrumentation.IsIgnored() {
		extra, err = m.defaultPackageInstrumentation.WriteExtraFiles()
		if err != nil {
			return nil, err
		}
	}

	if ht, err := m.writeHookTable(); err != nil {
		return nil, err
	} else if ht != "" {
		extra = append(extra, ht)
	}

	return extra, nil
}

func (m *mainPackageInstrumentation) writeHookTable() (string, error) {
	// Create the hook table and compile it.
	// Get the full list of hooks
	hooks, err := readHookListFile(m.hookListFilepath)
	if err != nil {
		return "", err
	}

	if len(hooks) == 0 {
		log.Printf("skipping hook table generation: the list of hooks is empty")
		return "", nil
	}

	// Create the hook table file into the package build directory
	hookTableFile, err := createHookTableFile(m.packageBuildDir)
	if err != nil {
		return "", err
	}
	defer hookTableFile.Close()
	log.Printf("creating the hook table for %d hooks into `%s`", len(hooks), hookTableFile.Name())
	if err := writeHookTable(hookTableFile, hooks); err != nil {
		return "", err
	}

	// Add it into the argument list to compile it
	return hookTableFile.Name(), nil
}

func createHookTableFile(dir string) (*os.File, error) {
	filename := filepath.Join(dir, "sqreen-hooktable.go")
	return os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
}

// Create or append the hook list file in write-only.
func openHookListFile(hookListFilepath string) (*os.File, error) {
	return os.OpenFile(hookListFilepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
}

func getHookListFilepath(dir string) string {
	return filepath.Join(dir, "sqreen-hooks.txt")
}

// Read the given hook list file by reopening it and reading its full content,
// returned as a slice of hook IDs.
func readHookListFile(hookListFilepath string) (hooks []string, err error) {
	f, err := os.OpenFile(hookListFilepath, os.O_RDONLY, 0666)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// Read each hook id line by line
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		id := scanner.Text()
		hooks = append(hooks, id)
	}
	return
}

type runtimePackageInstrumentation struct {
	packageInstrumentationHelper
	packageBuildDir string
}

func newRuntimePackageInstrumentation(packageBuildDir string) *runtimePackageInstrumentation {
	return &runtimePackageInstrumentation{
		packageBuildDir: packageBuildDir,
	}
}

func (runtimePackageInstrumentation) IsIgnored() bool {
	// This instrumentation is never ignored
	return false
}

func (h *runtimePackageInstrumentation) Instrument() (instrumented []*dst.File, err error) {
	v := newRuntimeInstrumentationVisitor()
	return h.packageInstrumentationHelper.instrument(v)
}

func (h *runtimePackageInstrumentation) WriteExtraFiles() ([]string, error) {
	rtExtensions := filepath.Join(h.packageBuildDir, "sqreen.go")
	if err := ioutil.WriteFile(rtExtensions, []byte(`package runtime
import _ "unsafe" // for go:linkname

//go:nosplit
//go:linkname _sqreen_gls_get _sqreen_gls_get
func _sqreen_gls_get() interface{} { return getg().m.curg.sqgls }

//go:nosplit
//go:linkname _sqreen_gls_set _sqreen_gls_set
func _sqreen_gls_set(v interface{})  { getg().m.curg.sqgls = v }
`), 0644); err != nil {
		return nil, err
	}
	return []string{rtExtensions}, nil
}

type runtimeInstrumentationVisitor struct {
	packageInstrumentationHelper
}

func newRuntimeInstrumentationVisitor() *runtimeInstrumentationVisitor {
	return &runtimeInstrumentationVisitor{}
}

func (v *runtimeInstrumentationVisitor) instrument(root *dst.Package) (instrumentedFiles []*dst.File) {
	var instrumented bool
	dstutil.Apply(root, func(cursor *dstutil.Cursor) bool {
		switch n := cursor.Node().(type) {
		default:
			return true

		case *dst.TypeSpec:
			if n.Name != nil && n.Name.Name != "g" {
				return true
			}
			st, ok := n.Type.(*dst.StructType)
			if !ok {
				return true
			}
			st.Fields.List = append(st.Fields.List, &dst.Field{
				Names: []*dst.Ident{dst.NewIdent("sqgls")},
				Type:  dst.NewIdent("interface{}"),
			})
			instrumented = true
			return true
		}
	},
		func(cursor *dstutil.Cursor) bool {
			if n, ok := cursor.Node().(*dst.File); ok && instrumented {
				instrumentedFiles = []*dst.File{n}
				return false
			}
			return true
		})
	return
}
