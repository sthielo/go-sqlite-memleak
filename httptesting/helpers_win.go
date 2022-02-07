//go:build windows
// +build windows

package httptesting

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func getProcessStats(t *testing.T) *ProcessStatEntry {
	cmd := exec.Command("tasklist", "/fo", "csv", "/nh", "/fi", "IMAGENAME eq oom.exe")
	out, err := cmd.Output()
	assert.Nilf(t, err, "could not execute `tasklist`: %+v", err)
	splits := strings.Split(string(out), ",")
	assert.Greaterf(t, len(splits), 4, "tasklist output not as expected: %s", string(out))
	mem := splits[4]

	cmd = exec.Command("cmd", "/C", "handle -s -p oom.exe -nobanner")
	out, err = cmd.Output()
	fdCnt := "n/a - needs `handle.exe` from `sysinternals`"
	if err != nil {
		_, _ = os.Stdout.WriteString(fmt.Sprintf("Could not execute `handle` (needs `sysinternals`to be installed) to gather file descriptor usage: %+v\n", err))
	} else {
		fdCnt = extractFileHandleCount(t, string(out))
	}
	return &ProcessStatEntry{mem, fdCnt}
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

func extractFileHandleCount(t *testing.T, s string) string {
	r := regexp.MustCompile(`(?ms).*^\s*File\s*:\s*([0-9]+)\s*$.*`)
	match := r.FindStringSubmatch(s)
	assert.Greaterf(t, len(match), 0, "output of `handle.exe` not as expected: %s", s)
	return match[1]
}
