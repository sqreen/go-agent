// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"errors"
	"log"
)

type compileFlagSet struct {
	Package string `sqflag:"-p"`
	Output  string `sqflag:"-o"`
}

func (f *compileFlagSet) Validate() error {
	if f.Package == "" {
		return errors.New("unexpected empty package option")
	}
	if f.Output == "" {
		return errors.New("unexpected empty output option")
	}
	return nil
}

func parseCompileCommand(args []string) (execCommandFunc, error) {
	if len(args) == 0 {
		return nil, errors.New("unexpected number of command arguments")
	}

	flags := &compileFlagSet{}
	parseFlags(flags, args[1:])
	if err := flags.Validate(); err != nil {
		return nil, err
	}

	return func() error {
		log.Println(flags)
		return nil
	}, nil
}
