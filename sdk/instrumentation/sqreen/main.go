// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("sqreen: ")

	cmd, err := parseCommand(os.Args[1:])
	if err != nil {
		log.Println(err)
		printUsage()
		os.Exit(1)
	}

	if err := cmd(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func printUsage() {
	const usageMessage = "printUsage: TODO\n"
	_, _ = fmt.Fprint(os.Stderr, usageMessage)
	os.Exit(2)
}

type parseCommandFunc func([]string) (execCommandFunc, error)
type execCommandFunc func() error

var commandParserMap = map[string]parseCommandFunc{
	//"instrument": parseInstrumentCmd,
	"compile": parseCompileCommand,
}

// getCommand returns the command and arguments. The command is expectedFlags to be
// the first argument.
func parseCommand(args []string) (execCommandFunc, error) {
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
	commandParser, exists := commandParserMap[cmdId]
	if !exists {
		return nil, fmt.Errorf("unexpected command `%s`", cmdId)
	}

	return commandParser(args)
}
