//go:build !windows
// +build !windows

package httptesting

import (
	"bufio"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func getProcessStats(t *testing.T) *ProcessStatEntry {
	cmd := exec.Command("ps", "-o", "pid,rss,vsz,drs,sz,comm", "-C", "oom", "--no-headers")
	out, err := cmd.Output()
	assert.Nilf(t, err, "Could not execute `ps` to gather mem usage: %+v", err)
	splits := strings.Split(string(out), " ")
	assert.Greaterf(t, len(splits), 2, "`ps` output not as expected (pid,rss,vsz,drs,sz,comm): %s", string(out))
	mem := splits[2]

	cmd = exec.Command("lsof", "-c", "oom")
	out, err = cmd.Output()
	assert.Nilf(t, err, "Could not execute `lsof` to gather file descriptor usage: %+v", err)

	fdCnt := countLines(string(out)) - 1 /* header row */
	return &ProcessStatEntry{mem, strconv.Itoa(fdCnt)}
}

func startMain(t *testing.T) (*exec.Cmd, io.ReadCloser, io.WriteCloser) {
	wd, _ := os.Getwd()
	testee := exec.Command("./artifacts/oom")
	testee.Stderr = os.Stderr
	childStdout, err := testee.StdoutPipe()
	assert.Nil(t, err, "error gettng testee's stdout pipe (%s in %s): %+v", testee.Path, wd, err)
	childStdin, err := testee.StdinPipe()
	assert.Nil(t, err, "error gettng testee's stdin pipe (%s in %s): %+v", testee.Path, wd, err)
	err = testee.Start()
	assert.Nil(t, err, "error starting testee (%s in %s): %+v", testee.Path, wd, err)
	return testee, childStdout, childStdin
}

func countLines(s string) int {
	scanner := bufio.NewScanner(strings.NewReader(s))
	counter := 0
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 {
			counter++
		}
	}
	return counter
}
