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

func getProcessMem() string {
	cmd := exec.Command("tasklist", "/fo", "csv", "/nh", "/fi", "IMAGENAME eq oom.exe")
	b, _ := cmd.Output()
	splits := strings.Split(string(b), ",")
	if len(splits) > 4 {
		return splits[4]
	}
	return string(b)
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
