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

		// Check if the instrumentation should be skipped for this package name.
		pkgPath := flags.Package
		if isPackageNameIgnored(pkgPath) {
			log.Printf("skipping instrumentation of package `%s`\n", pkgPath)
			return nil, nil
		}

		packageBuildDir := filepath.Dir(flags.Output)

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
			// to later change it if it gets instrumented.
			argEntries[src] = i
		}

		instrumented, err := pkgInstrumentation.instrument(pkgPath)
		if err != nil {
			return nil, err
		}
		if !instrumented {
			return args, nil
		}

		written, err := pkgInstrumentation.writeInstrumentedFiles(packageBuildDir)
		if err != nil {
			return nil, err
		}

		// Update the argument list by replacing source files that were
		// instrumented.
		for src, dest := range written {
			argIndex := argEntries[src]
			args[argIndex] = dest
		}

		// Add the hook IDs to the hook list file.
		projectBuildDir := path.Join(packageBuildDir, "..")
		hookListFile, err := openHookListFile(projectBuildDir)
		if err != nil {
			return nil, err
		}
		defer hookListFile.Close()

		count, err := pkgInstrumentation.writeHookList(hookListFile)
		if err != nil {
			return nil, err
		}
		log.Printf("added %d hooks to the hook list\n", count)

		return args, nil
	}
}

func openHookListFile(dir string) (*os.File, error) {
	filename := filepath.Join(dir, "sqreen-hooks.txt")
	// Create the file or append to it if it exists.
	return os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
}

type packageInstrumentationHelper struct {
	parsedFiles       map[string]*dst.File
	parsedFileSources map[*dst.File]string
	fset              *token.FileSet
	instrumentedFiles map[*dst.File][]*hookpoint
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

func (h *packageInstrumentationHelper) instrument(pkgPath string) (instrumented bool, err error) {
	if len(h.parsedFiles) == 0 {
		log.Println("nothing to instrument")
		return false, nil
	}

	root, err := dst.NewPackage(h.fset, h.parsedFiles, nil, nil)
	if err != nil {
		return false, err
	}

	v := newInstrumentationVisitor(pkgPath)
	h.instrumentedFiles = v.instrument(root)
	return len(h.instrumentedFiles) > 0, nil
}

func (h *packageInstrumentationHelper) writeInstrumentedFiles(buildDirPath string) (srcdst map[string]string, err error) {
	srcdst = make(map[string]string, len(h.instrumentedFiles))
	for node := range h.instrumentedFiles {
		src := h.parsedFileSources[node]
		filename := filepath.Base(src)
		dest := filepath.Join(buildDirPath, filename)
		output, err := os.Create(dest)
		if err != nil {
			log.Fatal(err)
		}
		defer output.Close()
		if err := writeFile(node, output); err != nil {
			return nil, err
		}
		srcdst[src] = dest
	}
	return srcdst, nil
}

func (h *packageInstrumentationHelper) writeHookList(hookList *os.File) (count int, err error) {
	for _, hooks := range h.instrumentedFiles {
		for _, hook := range hooks {
			if _, err = hookList.WriteString(fmt.Sprintf("%s\n", hook.id)); err != nil {
				return count, err
			}
			count += 1
		}
	}
	return count, nil
}
