package database

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/thanhpk/randstr"
	"math/rand"
	"os"
	"time"
)

func FillInDummyData() {
	nrInT1 := 2500
	nrInT10 := 30
	nrInT11 := 1800

	start := time.Now()
	_, _ = os.Stdout.WriteString(">>> oom: " + ("starting to fill in dummy data ...") + "\n")

	t10Vals := make([]string, 0, nrInT10+1)
	t10Vals = append(t10Vals, "aaa")

	prepareInsT10, err := MyDb.Prepare(`insert into t10 (id, t10f1, t10f2) values (?,'aaa',?)`)
	failOnErr("failed prepareInsT10", err)
	defer prepareInsT10.Close()

	prepareInsT11, err := MyDb.Prepare(`insert into t11 (t10_id, t11f1, t11f2) values (?,?,?)`)
	failOnErr("failed prepareInsT11", err)
	defer prepareInsT11.Close()

	for i := 0; i < nrInT10; i++ {
		t10Id := genUuid()
		t10f2 := genString(3, alphaNumeric, t10Vals)
		t10Vals = append(t10Vals, t10f2)
		_, err = prepareInsT10.Exec(t10Id, t10f2)
		failOnErr("failed to execute prepareInsT10", err)

		for j := 0; j < nrInT11; j++ {
			t11f1 := genDate(-j)
			t11f2 := genFloat()
			_, err = prepareInsT11.Exec(t10Id, t11f1, t11f2)
			failOnErr("failed to execute prepareInsT11", err)
		}
	}

	prepareInsT1, err := MyDb.Prepare(`insert into t1 (id, t1f1, t1f2, t1f3) values (?,?,1,?)`)
	failOnErr("failed prepareInsT1", err)
	defer prepareInsT1.Close()

	prepareInsT2, err := MyDb.Prepare(`insert into t2 (id, t2f1, t2f2) values (?,?,?)`)
	failOnErr("failed prepareInsT2", err)
	defer prepareInsT2.Close()

	prepareInsT3, err := MyDb.Prepare(`insert into t3 (id, t3f1, t3f2, t3f3, t3f4, t3f5, t3f8) values (?,?,?,?,12,?,?)`)
	failOnErr("failed prepareInsT3", err)
	defer prepareInsT3.Close()

	prepareInsT5, err := MyDb.Prepare(`insert into t5 (id, t1_id, t5f1, t5f2, t5f3, t5f4) values (?,?,1,?,?,'valC')`)
	failOnErr("failed prepareInsT5", err)
	defer prepareInsT5.Close()

	prepareInsT6, err := MyDb.Prepare(`insert into t6 (t5_id, t6f1, t6f2) values (?,?,?)`)
	failOnErr("failed prepareInsT6", err)
	defer prepareInsT6.Close()

	t2Vals := make([]string, 0, 2*nrInT1)

	lastUsageStr := genDate(-5)
	for i := 0; i < nrInT1; i++ {
		t1Id := genUuid()
		t1f1 := genString(100, allCharsSpacesLineBreaks, nil)
		_, err = prepareInsT1.Exec(t1Id, t1f1, lastUsageStr)
		failOnErr("failed to execute prepareInsT1", err)

		t2f2 := genString(13, alphaNumeric, t2Vals)
		t2Vals = append(t2Vals, t2f2)
		_, err = prepareInsT2.Exec(t1Id, "valA", t2f2)
		failOnErr("failed to execute prepareInsT2-1", err)
		t2f2 = genString(12, alphaNumeric, t2Vals)
		t2Vals = append(t2Vals, t2f2)
		_, err = prepareInsT2.Exec(t1Id, "valB", t2f2)
		failOnErr("failed to execute prepareInsT2-2", err)

		t3f1 := genString(24, allChars, nil)
		t3f2 := genString(24, allChars, nil)
		t3f3 := genDate(10 + i)
		t3f5 := genFloat()
		t3f8 := genString(8, alphaNumeric, nil)
		_, err = prepareInsT3.Exec(t1Id, t3f1, t3f2, t3f3, t3f5, t3f8)
		failOnErr("failed to execute prepareInsT3", err)

		t5Id := genUuid()
		t5f2 := genDate(-i)
		_, err = prepareInsT5.Exec(t5Id, t1Id, t5f2, t2f2)
		failOnErr("failed to execute prepareInsT5", err)

		for j := 0; j < nrInT11; j++ {
			t6f1 := genDate(-j)
			t6f2 := genFloat()
			_, err = prepareInsT6.Exec(t5Id, t6f1, t6f2)
			failOnErr("failed to execute prepareInsT6", err)
		}
	}

	_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf("creating dummy data took: %.0fs", time.Since(start).Seconds()) + "\n")
}

func failOnErr(msg string, err error) {
	if err != nil {
		_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf(msg+": %+v", err) + "\n")
		os.Exit(1)
	}
}

var alphaNumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "abcdefghijklmnopqrstuvwxyz" + "0123456789"
var allChars = alphaNumeric + "?! _-:;.,#*$(){}[]^'+%&/"
var allCharsSpacesLineBreaks = allChars + " \t\n\r"

func genUuid() string {
	return uuid.New().String()
}

func genDate(i int) string {
	start := time.Now()
	someDate := start.AddDate(0, 0, i)
	return someDate.Format("2006-01-02")
}

func genFloat() float64 {
	return rand.Float64()
}

func genString(len int, letters string, exclusions []string) string {
	candidate := randstr.String(len, letters)
	if exclusions == nil || !containsString(exclusions, candidate) {
		return candidate
	}
	return genString(len, letters, exclusions)
}

func containsString(s []string, v string) bool {
	for _, vv := range s {
		if vv == v {
			return true
		}
	}
	return false
}
