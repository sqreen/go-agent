// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"errors"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

type compileFlagSet struct {
	Package string `sqflag:"-p"`
	Output  string `sqflag:"-o"`
}

func (f *compileFlagSet) IsValid() bool {
	return f.Package != "" && f.Output != ""
}

func (f *compileFlagSet) String() string {
	return fmt.Sprintf("-p=%q -o=%q", f.Package, f.Output)
}

func parseCompileCommand(args []string) (commandExecutionFunc, error) {
	if len(args) == 0 {
		return nil, errors.New("unexpected number of command arguments")
	}
	flags := &compileFlagSet{}
	parseFlags(flags, args[1:])
	return makeCompileCommandExecutionFunc(flags, args), nil
}

func makeCompileCommandExecutionFunc(flags *compileFlagSet, args []string) commandExecutionFunc {
	return func() ([]string, error) {
		log.Println("compile")
		if !flags.IsValid() {
			// Skip when the required set of flags is not valid.
			log.Printf("nothing to do (%s)\n", flags)
			return nil, nil
		}

		// Check if the instrumentation should be skipped for this package name
		pkgPath := flags.Package
		if isPackageNameIgnored(pkgPath) {
			log.Printf("skipping instrumentation of package `%s`\n", pkgPath)
			return nil, nil
		}

		packageBuildDir := filepath.Dir(flags.Output)
		//buildDir := path.Join(packageBuildDir, "..")

		log.Println("instrumenting package:", pkgPath)
		log.Println("package build directory:", packageBuildDir)

		// Make the list of Go files to instrument out of the argument list and
		// replace their argument list entry by their instrumented copy.
		var pkgInstrumentation packageInstrumentationHelper
		for i, src := range args {
			// Only consider args ending with the Go file extension.
			if !strings.HasSuffix(src, ".go") {
				continue
			}

			// Check if the instrumentation should be skipped for this filename
			basename := filepath.Base(src)
			if isFileNameIgnored(basename) {
				log.Println("skipping instrumentation of file", src)
				continue
			}

			// Put the instrumented file into the package compilation directory.
			dest := filepath.Join(packageBuildDir, basename)
			// Add it to the list of files to instrument.
			ignored, err := pkgInstrumentation.addFile(src, dest)
			if err != nil {
				return nil, err
			}
			if !ignored {
				// Replace the argument list entry with the instrumented file name.
				args[i] = dest
			}
		}

		err := pkgInstrumentation.instrument(pkgPath, packageBuildDir)
		if err != nil {
			return nil, err
		}

		//// If any, add the extra metadata file to the argument list
		//if metadataFilepath != "" {
		//	args = append(args, metadataFilepath)
		//}
		return args, nil
	}
}

type packageInstrumentationHelper struct {
	toInstrument map[*dst.File]instrumentationFiles
	fset         *token.FileSet
}

// addFile parses the given Go source file `src` and adds it to the set of
// files to instrument if it is not ignored by a directive.
func (h *packageInstrumentationHelper) addFile(src string, dest string) (ignored bool, err error) {
	log.Printf("parsing file `%s`", src)
	if h.fset != nil {
		// The token fileset is required to later create the package node.
		h.fset = token.NewFileSet()
	}
	file, err := decorator.ParseFile(h.fset, src, nil, parser.ParseComments)
	if err != nil {
		return true, err
	}
	if hasSqreenIgnoreDirective(file) {
		log.Printf("file `%s` skipped due to ignore directive", src)
		return true, nil
	}
	if h.toInstrument == nil {
		h.toInstrument = make(map[*dst.File]instrumentationFiles)
	}
	h.toInstrument[file] = instrumentationFiles{src: src, dst: dest}
	return false, nil
}

func (h *packageInstrumentationHelper) instrument(pkgPath string, buildDir string) error {
	if len(h.toInstrument) == 0 {
		log.Println("nothing to instrument")
		return nil
	}

	files := make(map[string]*dst.File, len(h.toInstrument))
	for node, d := range h.toInstrument {
		files[d.src] = node
	}

	root, err := dst.NewPackage(h.fset, files, nil, nil)
	if err != nil {
		return err
	}

	//pkgName := root.Name
	//log.Println("creating instrumentation metadata file")
	//metadataFile, err := createMetadataFile(pkgName, buildDir)
	//if err != nil {
	//	log.Println("could not create the package instrumentation metadata file")
	//	return err
	//}
	//defer metadataFile.Close()

	v := newInstrumentationVisitor(pkgPath)
	v.instrument(root)

	// TODO: May be not instrumented at all, so we would want to keep the original one...
	//   rather return a map to new locations
	if err := h.writeInstrumentedFiles(); err != nil {
		return err
	}

	return nil
}

func (h *packageInstrumentationHelper) writeInstrumentedFiles() error {
	for node, d := range h.toInstrument {
		fset, af, err := decorator.RestoreFile(node)
		if err != nil {
			return err
		}
		output, err := os.Create(d.dst)
		if err != nil {
			log.Fatal(err)
		}
		defer output.Close()
		if err := printer.Fprint(output, fset, af); err != nil {
			return err
		}
	}
	return nil
}

func ignoreFuncDecl(funcDecl *dst.FuncDecl) bool {
	return funcDecl.Name.Name == "_" ||
		funcDecl.Name.Name == "init" ||
		funcDecl.Body == nil ||
		hasSqreenIgnoreDirective(funcDecl) ||
		hasGoNoSplitDirective(funcDecl) ||
		functionScopeHidesSignatureTypes(funcDecl, dst.NewIdent("error"))
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

func functionScopeHidesSignatureTypes(fdecl *dst.FuncDecl, extraType ...dst.Expr) (collision bool) {
	ftype := fdecl.Type
	idents := []string{fdecl.Name.Name}
	ftypes := extraType
	if recv := fdecl.Recv; recv != nil {
		for _, p := range recv.List {
			for _, n := range p.Names {
				idents = append(idents, n.Name)
			}
			ftypes = append(ftypes, p.Type)
		}
	}
	if ftype.Params != nil {
		for _, p := range ftype.Params.List {
			for _, n := range p.Names {
				idents = append(idents, n.Name)
			}
			ftypes = append(ftypes, p.Type)
		}
	}
	if ftype.Results != nil {
		for _, p := range ftype.Results.List {
			for _, n := range p.Names {
				idents = append(idents, n.Name)
			}
			ftypes = append(ftypes, p.Type)
		}
	}

	checkScope := func(node dst.Node) bool {
		if ident, ok := node.(*dst.Ident); ok {
			for _, scopeIdent := range idents {
				if scopeIdent == ident.Name || scopeIdent == ident.Path {
					collision = true
					return false
				}
			}
		}
		return true
	}

	for i := range ftypes {
		dst.Inspect(ftypes[i], checkScope)
		if collision {
			break
		}
	}
	return
}

type instrumentationFiles struct {
	src, dst string
}

func createMetadataFile(pkgName string, compilationDir string) (*os.File, error) {
	const packageInstrumentationMetadataFileName = "_sqreen_metadata_.go"
	f, err := os.Create(filepath.Join(compilationDir, packageInstrumentationMetadataFileName))
	if err != nil {
		return nil, err
	}

	if _, err := f.WriteString(fmt.Sprintf("package %s\n\n", path.Base(pkgName))); err != nil {
		_ = f.Close()
		return nil, err
	}

	return f, nil
}

func endPackageInstrumentation(*os.File) {
	// TODO: add required types and functions
}

func isFileNameIgnored(filename string) bool {
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

func isPackageNameIgnored(pkg string) bool {
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

	for _, prefix := range ignoredPkgPrefixes {
		if strings.HasPrefix(pkg, prefix) {
			return true
		}
	}

	return false
}
