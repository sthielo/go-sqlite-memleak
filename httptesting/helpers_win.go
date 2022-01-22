//go:build windows
// +build windows

package httptesting

import (
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func getProcessMem(t *testing.T) *ProcessStatEntry {
	cmd := exec.Command("tasklist", "/fo", "csv", "/nh", "/fi", "IMAGENAME eq oom.exe")
	b, err := cmd.Output()
	assert.Nilf(t, err, "could not execute `tasklist`: %+v", err)
	splits := strings.Split(string(b), ",")
	assert.Greaterf(t, len(splits), 4, "tasklist output not as expected: %s", string(b))
	return &ProcessStatEntry{splits[4], "n/a"}
}

func startMain(t *testing.T) *exec.Cmd {
	wd, _ := os.Getwd()
	testee := exec.Command("./artifacts/oom.exe")
	testee.Stdout = os.Stdout
	testee.Stderr = os.Stderr
	err := testee.Start()
	assert.Nil(t, err, "error starting testee (%s in %s): %+v", testee.Path, wd, err)
	return testee
}
