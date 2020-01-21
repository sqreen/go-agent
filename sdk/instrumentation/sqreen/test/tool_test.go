// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqreen_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstrumentation(t *testing.T) {
	toolPath := buildInstrumentationTool(t)
	defer os.Remove(toolPath)
	//packagestest.TestAll(t, testInstrumentation(toolPath))
	myTest(t, toolPath)
}

func buildInstrumentationTool(t *testing.T) (path string) {
	toolFile, err := ioutil.TempFile("", "sqreen-instrumentation")
	require.NoError(t, err)
	toolFile.Close()
	toolPath := toolFile.Name()
	goroot := os.Getenv("GOROOT")
	if goroot != "" {
		goroot += "/bin/"
	}
	cmd := exec.Command(goroot+"go", "build", "-o", toolPath, "github.com/sqreen/go-agent/sdk/instrumentation/sqreen")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	require.NoError(t, err)
	return toolPath
}

func myTest(t *testing.T, toolPath string) {
	goroot := os.Getenv("GOROOT")
	if goroot != "" {
		goroot += "/bin/"
	}
	cmd := exec.Command(goroot+"go", "run", "-a", "-toolexec", toolPath, "./testdata/hello-world")
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	require.NoError(t, err)

	expectedOutput, err := ioutil.ReadFile("./testdata/hello-world/output.txt")
	require.NoError(t, err)
	require.Equal(t, expectedOutput, output)
}
