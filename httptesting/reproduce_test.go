package httptesting

import (
	"bufio"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
	"time"
)

const totalRuns = 20 // => adjust test timeout in Makefile !!!

var cmdDumpDb = "DUMP"
var cmdSnapshot = "SNAPSHOT"
var cmdEnd = "END"

func TestReproduceOoM(t *testing.T) {
	_ = os.Chdir("..")
	testee, childStdout, childStdin := startMain(t)
	defer childStdout.Close()
	defer childStdin.Close()
	childOutReader := bufio.NewReader(childStdout)

	waitForTestee(t, childOutReader)
	gatherProcStats(t)
	for r := 0; r < totalRuns; r++ {
		_, _ = os.Stdout.WriteString(fmt.Sprintf("starting run: %d\n", r))
		_ = os.Stdout.Sync()

		_, _ = childStdin.Write([]byte(cmdDumpDb + "\n"))
		//childStdin.Write([]byte(cmdSnapshot + "\n"));

		waitForTestee(t, childOutReader)
		time.Sleep(2 * time.Second) // allow garbage collection to happen
		gatherProcStats(t)
	}
	printProcStats()
	_, _ = childStdin.Write([]byte(cmdEnd + "\n"))

	childStdout.Close()
	childStdin.Close()
	_ = testee.Wait()
}

var memStats = make([]*ProcessStatEntry, 0, totalRuns+1)

func gatherProcStats(t *testing.T) {
	ps := getProcessStats(t)
	memStats = append(memStats, ps)
}

func printProcStats() {
	for i, m := range memStats {
		_, _ = os.Stdout.WriteString(fmt.Sprintf("    %d: %+v\n", i, m))
	}
	_ = os.Stdout.Sync()
}

// waiting for "DONE*" ...
func waitForTestee(t *testing.T, scanner *bufio.Reader) {
	done := false
	for !done {
		input, err := scanner.ReadString('\n')
		if err != nil {
			assert.Failf(t, "Failed to read child stdout", "%+v", err)
		}
		done = strings.HasPrefix(input, "DONE")
		_, _ = os.Stderr.WriteString("### oom-stdout: " + input + "\n")
	}
}
