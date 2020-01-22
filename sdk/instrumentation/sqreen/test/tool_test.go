// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqreen_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstrumentation(t *testing.T) {
	toolPath := buildInstrumentationTool(t)
	defer os.Remove(toolPath)
	myTest(t, toolPath)
}

func buildInstrumentationTool(t *testing.T) (path string) {
	toolDir, err := ioutil.TempDir("", "test-sqreen-instrumentation")
	require.NoError(t, err)
	toolPath := filepath.Join(toolDir, "sqreen")
	if runtime.GOOS == "windows" {
		toolPath += ".exe"
	}
	cmd := exec.Command(godriver, "build", "-o", toolPath, "github.com/sqreen/go-agent/sdk/instrumentation/sqreen")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	require.NoError(t, err)
	return toolPath
}

func myTest(t *testing.T, toolPath string) {
	cmd := exec.Command(godriver, "run", "-a", "-toolexec", toolPath, "./testdata/hello-world")
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	require.NoError(t, err)

	// Check that we got the expected execution output in stdout.
	expectedOutput, err := ioutil.ReadFile("./testdata/hello-world/output.txt")
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
