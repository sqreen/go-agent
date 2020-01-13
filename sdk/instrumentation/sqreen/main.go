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
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("sqreen: ")

	args := os.Args[1:]
	cmd, err := parseCommand(args)
	if err != nil {
		log.Println(err)
		printUsage()
		os.Exit(1)
	}

	if cmd != nil {
		// The command is implemented
		if err := cmd(); err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}

	forwardCommand(args)
}

// forwardCommand runs the given command's argument list and exits the process
// with the exit code that was returned.
func forwardCommand(args []string) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Println("forwarding command", cmd)
	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	os.Exit(0)
}

func printUsage() {
	const usageMessage = "printUsage: TODO\n"
	_, _ = fmt.Fprint(os.Stderr, usageMessage)
	os.Exit(2)
}

type parseCommandFunc func([]string) (commandExecutionFunc, error)
type commandExecutionFunc func() error

var commandParserMap = map[string]parseCommandFunc{
	//"instrument": parseInstrumentCmd,
	"compile": parseCompileCommand,
}

// getCommand returns the command and arguments. The command is expectedFlags to be
// the first argument.
func parseCommand(args []string) (commandExecutionFunc, error) {
	// At least one arg is expectedFlags
	if len(args) < 1 {
		return nil, errors.New("unexpected number of arguments")
	}
	cmdId := args[0]

	// It mustn't be empty
	if cmdId == "" {
		return nil, errors.New("unexpected empty command name")
	}

	// It may be the absolute path of a go tool: take its base name.
	cmdId = filepath.Base(cmdId)
	if commandParser, exists := commandParserMap[cmdId]; exists {
		return commandParser(args)
	} else {
		return nil, nil
	}
}
