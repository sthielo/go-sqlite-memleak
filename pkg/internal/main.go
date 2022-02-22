package main

import (
	"fmt"
	"github.com/sthielo/go-sqlite-memleak/pkg/internal/database"
	"os"
)

func main() {
	database.InitDB()
	defer database.MyDb.Close()
	database.FillInDummyData()

	_, _ = os.Stdout.WriteString("DONE\n")

	cmd := waitInput() // wait 'END' or 'CONTINUE'/'DUMP' or 'SNAPSHOT' (any input) - give time to gather process stats
	for i := 0; cmd != "END" && i < 30; i++ {

		if cmd == "CONTINUE" || cmd == "DUMP" {
			dumpDb()
		} else if cmd == "SNAPSHOT" {
			snapshotOnly()
		}

		_, _ = os.Stdout.WriteString(fmt.Sprintf("DONE iteration %d\n"))
		cmd = waitInput()
	}
}

func dumpDb() {
	err := database.Activity(database.ActivityDump)
	if err != nil {
		_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf("error when dumping db: %+v\n", err) + "\n")
		os.Exit(1)
	}
}

func snapshotOnly() {
	err := database.Activity(database.ActivityNone)
	if err != nil {
		_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf("error when dumping db: %+v\n", err) + "\n")
		os.Exit(1)
	}
}

func waitInput() string {
	var cmd string
	_, _ = fmt.Scanln(&cmd)
	return cmd
}
