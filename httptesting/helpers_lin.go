//go:build !windows
// +build !windows

package httptesting

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func getProcessMem() string {
	cmd := exec.Command("ps", "-o", "pid,rss,vsz,drs,sz,comm", "-C", "oom", "--no-headers")
	out, err := cmd.Output()
	if err != nil {
		_, _ = os.Stdout.WriteString(fmt.Sprintf("--- ERR: %+v\n", err))
		os.Exit(99)
	}
	splits := strings.Split(string(out), " ")
	if len(splits) > 2 {
		return splits[2] // rss
	}
	return string(out)
}

func startMain(t *testing.T) *exec.Cmd {
	wd, _ := os.Getwd()
	testee := exec.Command("./artifacts/oom")
	testee.Stdout = os.Stdout
	testee.Stderr = os.Stderr
	err := testee.Start()
	assert.Nil(t, err, "error starting testee (%s in %s): %+v", testee.Path, wd, err)
	return testee
}
