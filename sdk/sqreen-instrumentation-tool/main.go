// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/xerrors"
)

var globalFlags instrumentationToolFlagSet

func main() {
	log.SetFlags(0)
	log.SetPrefix("sqreen: ")
	log.SetOutput(os.Stderr)

	args := os.Args[1:]
	cmd, cmdArgPos, err := parseCommand(&globalFlags, args)
	if err != nil || globalFlags.Help {
		log.Println(err)
		printUsage()
		os.Exit(1)
	}

	// Don't passing instrumentation tool arguments
	if cmdArgPos != -1 {
		args = args[cmdArgPos:]
	}

	var logs strings.Builder
	if !globalFlags.Verbose {
		// Save the logs to show them in case of instrumentation error
		log.SetOutput(&logs)
	}

	if cmd != nil {
		// The command is implemented
		newArgs, err := cmd()
		if err != nil {
			log.Println(err)
			if !globalFlags.Verbose {
				fmt.Fprintln(os.Stderr, &logs)
			}
			os.Exit(1)
		}
		if newArgs != nil {
			// Args are replaced
			args = newArgs
		}
	}

	err = forwardCommand(args)
	var exitErr *exec.ExitError
	if xerrors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	os.Exit(0)
}

// forwardCommand runs the given command's argument list and exits the process
// with the exit code that was returned.
func forwardCommand(args []string) error {
	path := args[0]
	args = args[1:]
	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	quotedArgs := fmt.Sprintf("%+q", args)
	log.Printf("forwarding command `%s %s`", path, quotedArgs[1:len(quotedArgs)-1])
	return cmd.Run()
}

func printUsage() {
	const usageFormat = `Usage: go {build,install,get,test} -a -toolexec '%s [-v] [-full]' PACKAGES...

Sqreen's instrumentation tool for Go. It instruments Go source code at
compilation time by adding hooks on every instrumented functions. The set of
instrumented functions is restricted to a few required packages by default but
can be fully performed on every package by using option -full. Low-level package
such as cgo or the runtime are never instrumented. The verbose output can be
enabled using option -v and gives some details about the instrumentation.
It is integrated into the go toolchain thanks to the option -toolexec and must
be used along with option -a so that every package is compiled no matter the
previously cached compilations.

Note that Sqreen's Go agent provides the instrumentation engine and therefore
needs to be imported by the compiled package to be able to link.

Options:
        -h
                Print this usage message.
        -v
                Verbose mode. Detailed logs will be printed by the tool.
        -full
                Perform a full instrumentation of the program.

To see the instrumented code, use the go option -work in order to keep the
build directory. It will contain every instrumented Go source file.

Limitations:
- Debugging an instrumented program is possible by using the Go option -work so
  that debuggers can find the instrumented Go source code into the build
  directory. Better debugging support will be added in the future.
`
	_, _ = fmt.Fprintf(os.Stderr, usageFormat, os.Args[0])
	os.Exit(2)
}

type parseCommandFunc func([]string) (commandExecutionFunc, error)
type commandExecutionFunc func() (newArgs []string, err error)

var commandParserMap = map[string]parseCommandFunc{
	"compile": parseCompileCommand,
}

// getCommand returns the command and arguments. The command is expectedFlags to be
// the first argument.
func parseCommand(instrToolFlagSet *instrumentationToolFlagSet, args []string) (commandExecutionFunc, int, error) {
	cmdIdPos := parseFlagsUntilFirstNonOptionArg(instrToolFlagSet, args)
	if cmdIdPos == -1 {
		return nil, cmdIdPos, errors.New("unexpected arguments")
	}
	cmdId := args[cmdIdPos]
	args = args[cmdIdPos:]
	cmdId, err := parseCommandID(cmdId)
	if err != nil {
		return nil, cmdIdPos, err
	}

	if commandParser, exists := commandParserMap[cmdId]; exists {
		cmd, err := commandParser(args)
		return cmd, cmdIdPos, err
	} else {
		return nil, cmdIdPos, nil
	}
}

func parseCommandID(cmd string) (string, error) {
	// It mustn't be empty
	if cmd == "" {
		return "", errors.New("unexpected empty command name")
	}

	// Take the base of the absolute path of the go tool
	cmd = filepath.Base(cmd)
	// Remove the file extension if any
	if ext := filepath.Ext(cmd); ext != "" {
		cmd = strings.TrimSuffix(cmd, ext)
	}
	return cmd, nil
}
