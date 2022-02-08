package httptesting

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"testing"
	"time"
)

const totalRuns = 20 // => adjust test timeout in Makefile !!!

const testeeBaseUrl = "http://localhost:8890"

var urlDumpDb = fmt.Sprintf("%s/dumpdb", testeeBaseUrl)
var urlSnapshotDb = fmt.Sprintf("%s/snpshotonly", testeeBaseUrl)

func TestReproduceOoM(t *testing.T) {
	_ = os.Chdir("..")
	testee := startMain(t)

	waitForTestee(t)

	gatherProcStats(t)
	for r := 0; r < totalRuns; r++ {
		_, _ = os.Stdout.WriteString(fmt.Sprintf("starting run: %d\n", r))
		_ = os.Stdout.Sync()

		callTestee(t, urlDumpDb)
		//callTestee(t, urlSnapshotDb)

		time.Sleep(5 * time.Second) // allow garbage collection to happen
		gatherProcStats(t)
	}
	printProcStats()

	_ = testee.Process.Kill()
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

var httpTestClient = http.Client{
	// waiting for a db dump of a large db can take minutes :-(
	Timeout: 10 * time.Minute,
}

func callTestee(t *testing.T, url string) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	assert.Nil(t, err, "Unexpected error creating request: %+v", err)
	response, err := httpTestClient.Do(req)
	assert.Nil(t, err, "Unexpected error executing request: %+v", err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

// waiting for dummy data to be created - duration system dependant
func waitForTestee(t *testing.T) {
	for {
		time.Sleep(10 * time.Second)
		req, err := http.NewRequest(http.MethodGet, testeeBaseUrl, nil)
		assert.Nil(t, err, "Unexpected error creating request: %+v", err)
		_, err = httpTestClient.Do(req)
		if err == nil {
			break
		}
	}
}
