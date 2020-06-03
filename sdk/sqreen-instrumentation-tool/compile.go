// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
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
		if !flags.IsValid() {
			// Skip when the required set of flags is not valid.
			log.Printf("nothing to do (%s)\n", flags)
			return nil, nil
		}

		pkgPath := flags.Package
		packageBuildDir := filepath.Dir(flags.Output)

		var i Instrumenter
		switch pkgPath {
		case "runtime":
			i = newRuntimePackageInstrumentation(packageBuildDir)
		case "main":
			i = newMainPackageInstrumentation(pkgPath, globalFlags.Full, packageBuildDir)
		default:
			i = newDefaultPackageInstrumentation(pkgPath, globalFlags.Full, packageBuildDir)
		}

		if i.IsIgnored() {
			log.Printf("skipping instrumentation of package `%s`\n", pkgPath)
			return nil, nil
		}
		return instrument(i, args, pkgPath, packageBuildDir)
	}
}

// Update the argument list by replacing source files that were instrumented.
func updateArgs(args []string, argIndices map[string]int, written map[string]string) {
	for src, dest := range written {
		argIndex := argIndices[src]
		args[argIndex] = dest
	}
}

// Walk the list of arguments and add the go source files and the arg slice
// index to returned map.
func parseCompileCommandArgs(args []string) map[string]int {
	goFiles := make(map[string]int)
	for i, src := range args {
		// Only consider args ending with the Go file extension and assume they
		// are Go files.
		if !strings.HasSuffix(src, ".go") {
			continue
		}
		// Save the position of the source file in the argument list
		// to later change it if it gets instrumented.
		goFiles[src] = i
	}
	return goFiles
}

func instrument(i Instrumenter, args []string, pkgPath, packageBuildDir string) ([]string, error) {
	log.Println("instrumenting package:", pkgPath)
	log.Println("package build directory:", packageBuildDir)

	// Make the list of Go files to instrument out of the argument list and
	// replace their argument list entry by their instrumented copy.
	argIndices := parseCompileCommandArgs(args)
	for src := range argIndices {
		if err := i.AddFile(src); err != nil {
			return nil, err
		}
	}

	if instrumented, err := i.Instrument(); err != nil {
		return nil, err
	} else if len(instrumented) > 0 {
		written, err := i.WriteInstrumentedFiles(packageBuildDir, instrumented)
		if err != nil {
			return nil, err
		}
		// Replace original files in the args by the new ones
		updateArgs(args, argIndices, written)
	}

	extraFiles, err := i.WriteExtraFiles()
	if err != nil {
		return nil, err
	}

	args = append(args, extraFiles...)
	return args, nil
}
