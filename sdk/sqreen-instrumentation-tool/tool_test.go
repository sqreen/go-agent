// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstrumentation(t *testing.T) {
	toolPath := buildInstrumentationTool(t)
	defer os.Remove(toolPath)
	t.Run("hello-world", func(t *testing.T) {
		testInstrumentation(t, toolPath, "./testdata/hello-world")
	})

	t.Run("hello-example", func(t *testing.T) {
		testInstrumentation(t, toolPath, "./testdata/hello-example")
	})

	t.Run("hello-gls", func(t *testing.T) {
		testInstrumentation(t, toolPath, "./testdata/hello-gls")
	})
}

func buildInstrumentationTool(t *testing.T) (path string) {
	toolDir, err := ioutil.TempDir("", "test-sqreen-instrumentation")
	require.NoError(t, err)
	toolPath := filepath.Join(toolDir, "sqreen")
	if runtime.GOOS == "windows" {
		toolPath += ".exe"
	}
	cmd := exec.Command(godriver, "build", "-o", toolPath, "github.com/sqreen/go-agent/sdk/sqreen-instrumentation-tool")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	require.NoError(t, err)
	return toolPath
}

func testInstrumentation(t *testing.T, toolPath string, testApp string) {
	// Run it with full instrumentation and verbose mode
	cmd := exec.Command(godriver, "run", "-a", "-toolexec", fmt.Sprintf("%s -v -full", toolPath), testApp)
	gocache, err := ioutil.TempDir("", "go-cache")
	require.NoError(t, err, "could not create a go-cache directory")
	cmd.Env = append(os.Environ(), "GOCACHE="+gocache)
	cmd.Stderr = os.Stderr
	outputBuf, err := cmd.Output()
	require.NoError(t, err)

	output := string(outputBuf)
	fmt.Print(output)

	// Check the expected output against the output.txt file if any
	outputFile := filepath.Join(testApp, "output.txt")
	if !fileExists(outputFile) {
		return
	}

	// Check that we got the expected execution outputBuf in stdout.
	expectedOutputBuf, err := ioutil.ReadFile(outputFile)
	require.NoError(t, err)
	expectedOutput := strings.ReplaceAll(string(expectedOutputBuf), "\r\n", "\n") // windows seems to change te file \n into \r\n
	require.NoError(t, err)
	require.Equal(t, expectedOutput, output)
}

var (
	goroot   string
	godriver string
)

func init() {
	// Use same GOROOT as `go test` if any.
	goroot = os.Getenv("GOROOT")
	godriver = gobinpath("go")
}

func gobinpath(tool string) string {
	if goroot == "" {
		return tool
	}
	return filepath.Join(goroot, "bin", tool)
}

func fileExists(f string) bool {
	info, err := os.Stat(f)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
