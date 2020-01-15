// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
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
		argEntries := make(map[string]int)
		for i, src := range args {
			// Only consider args ending with the Go file extension and assume they
			// are Go files.
			if !strings.HasSuffix(src, ".go") {
				continue
			}
			if err := pkgInstrumentation.addFile(src); err != nil {
				return nil, err
			}
			// Save the position of the source file in the argument list
			// to later change it if it gets instrumented
			argEntries[src] = i
		}

		instrumented, err := pkgInstrumentation.instrument(pkgPath)
		if err != nil {
			return nil, err
		}

		for src, node := range instrumented {
			basename := filepath.Base(src)
			dest := filepath.Join(packageBuildDir, basename)
			output, err := os.Create(dest)
			if err != nil {
				log.Fatal(err)
			}
			defer output.Close()
			if err := writeFile(node, output); err != nil {
				return nil, err
			}
			argIndex := argEntries[src]
			args[argIndex] = dest
		}

		return args, nil
	}
}

type packageInstrumentationHelper struct {
	parsedFiles       map[string]*dst.File
	parsedFileSources map[*dst.File]string
	fset              *token.FileSet
}

// addFile parses the given Go source file `src` and adds it to the set of
// files to instrument if it is not ignored by a directive.
func (h *packageInstrumentationHelper) addFile(src string) error {
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

func (h *packageInstrumentationHelper) instrument(pkgPath string) (map[string]*dst.File, error) {
	if len(h.parsedFiles) == 0 {
		log.Println("nothing to instrument")
		return nil, nil
	}

	root, err := dst.NewPackage(h.fset, h.parsedFiles, nil, nil)
	if err != nil {
		return nil, err
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
	instrumentedFiles := v.instrument(root)

	// Return the map of source filepaths and instrumented ASTs
	instrumented := make(map[string]*dst.File, len(instrumentedFiles))
	for _, file := range instrumentedFiles {
		src := h.parsedFileSources[file]
		instrumented[src] = file
	}
	return instrumented, nil
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
	//functionScopeHidesSignatureTypes(funcDecl, dst.NewIdent("error"))
}

//func functionScopeHidesSignatureTypes(fdecl *dst.FuncDecl, extraType ...dst.Expr) (collision bool) {
//	ftype := fdecl.Type
//	idents := []string{fdecl.Name.Name}
//	ftypes := extraType
//	if recv := fdecl.Recv; recv != nil {
//		for _, p := range recv.List {
//			for _, n := range p.Names {
//				idents = append(idents, n.Name)
//			}
//			ftypes = append(ftypes, p.Type)
//		}
//	}
//	if ftype.Params != nil {
//		for _, p := range ftype.Params.List {
//			for _, n := range p.Names {
//				idents = append(idents, n.Name)
//			}
//			ftypes = append(ftypes, p.Type)
//		}
//	}
//	if ftype.Results != nil {
//		for _, p := range ftype.Results.List {
//			for _, n := range p.Names {
//				idents = append(idents, n.Name)
//			}
//			ftypes = append(ftypes, p.Type)
//		}
//	}
//
//	checkScope := func(node dst.Node) bool {
//		if ident, ok := node.(*dst.Ident); ok {
//			for _, scopeIdent := range idents {
//				if scopeIdent == ident.Name || scopeIdent == ident.Path {
//					collision = true
//					return false
//				}
//			}
//		}
//		return true
//	}
//
//	for i := range ftypes {
//		dst.Inspect(ftypes[i], checkScope)
//		if collision {
//			break
//		}
//	}
//	return
//}
