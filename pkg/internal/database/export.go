package database

import (
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"github.com/schollz/sqlite3dump"
	"os"
	"time"
)

/*
 * as dumping a db to sql stmts is a fairly slow process, the export first makes an in-memory backup (snapshot) of the
 * database, which can then be dumped without blocking the main db for regular usage (e.g. UI requests)
 * as snapshotting is non-invasive, meaning it offers time slots for requests to happen, it may fail when such requests
 * update/change the database => export could be retried ... done so in our productive project
 */
func Export() error {
	_, _ = os.Stdout.WriteString(">>> oom: " + ("starting export ...\n") + "\n")
	err := withSnapshotDo(func(dbToBackup *sql.DB) error {
		return dumpToFile(dbToBackup) // snapshot db activity
		// DISABLE snapshot db activity: return nil
	})
	if err != nil {
		_, _ = os.Stdout.WriteString(">>> oom: " + ("failed to dump db") + "\n")
		return err
	}

	// ??? PRAGMA shrink_memory: does not seem to have any impact on memory growth observation
	//_, err = MyDb.Exec("PRAGMA shrink_memory")
	//if err != nil {
	//	log.Log(fmt.Sprintf("failed to shrink db memory: %+v", err))
	//}

	_, _ = os.Stdout.WriteString(">>> oom: " + ("done export\n") + "\n")
	return nil
}

func dumpToFile(dbToBackup *sql.DB) error {
	ts := time.Now().Format("20060102150405")
	dumpfile, err := os.CreateTemp("tmp", fmt.Sprintf("dump-%s-*.sql.gz", ts))
	if err != nil {
		return err
	}
	defer dumpfile.Close()

	gw := gzip.NewWriter(dumpfile)
	defer gw.Close()

	err = sqlite3dump.DumpDB(dbToBackup, gw, sqlite3dump.WithMigration())
	return err
}

func withSnapshotDo(exec func(snapshot *sql.DB) error) error {
	file, err := os.CreateTemp("tmp", ".snapshot-*.db")
	if err != nil {
		_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf("cannot create temporary snapshotDb file - err: %+v", err) + "\n")
	}
	_ = file.Close()
	defer func() {
		err2 := os.RemoveAll(file.Name())
		if err2 != nil {
			_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf("failed to remove temp snapshot db file %s", file.Name()) + "\n")
		}
	}()

	snapshotConnStr := fmt.Sprintf("file:%s?mode=memory&cache=private&_journal_mode=OFF&_fk=off&_query_only=true&_locking=EXCLUSIVE&_mutex=no", file.Name())
	// next line is a WORKAROUND - using file based db instead of in-mem (mode=rwc):
	// snapshotConnStr := fmt.Sprintf("file:%s?mode=rwc&cache=private&_journal_mode=OFF&_fk=off&_query_only=true&_locking=EXCLUSIVE&_mutex=no", file.Name())
	snapshotDb, err := sql.Open("sqlite3", snapshotConnStr)
	if err != nil {
		_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf("cannot create snapshotDb - err: %+v", err) + "\n")
		return err
	}
	defer snapshotDb.Close()

	snapshotDb.SetMaxOpenConns(1)

	err = withSqliteConnDo(snapshotDb, func(snapshotSqliteConn *sqlite3.SQLiteConn) error {
		return withSqliteConnDo(MyDb, func(srcSqliteConn *sqlite3.SQLiteConn) error {
			return createDbSnapshot(snapshotSqliteConn, srcSqliteConn)
		})
	})

	if err == nil {
		err = exec(snapshotDb)
	}

	return err
}

/*
 * an accessor to the sqlite3 driver's native connection implementation
 */
func withSqliteConnDo(db *sql.DB, exec func(sqliteConn *sqlite3.SQLiteConn) error) error {
	connCtx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	conn, err := db.Conn(connCtx)
	if err != nil {
		_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf("failed to get driverConn - err: %+v", err) + "\n")
		return err
	}
	defer conn.Close()

	return conn.Raw(func(driverConn interface{}) error {
		sqliteConn := driverConn.(*sqlite3.SQLiteConn)
		return exec(sqliteConn)
	})
}

/*
 * snapshotting uses sqlite's Backup API to copy chunk of pages. the chunk size is limited, not to block the src db for to long.
 * when a db was updated between copying two succeeding chunks, the snapshotting fails => triggers retry mechanism in our production project
 */
func createDbSnapshot(snaphshotSqliteConn *sqlite3.SQLiteConn, srcSqliteConn *sqlite3.SQLiteConn) error {
	backup, err := snaphshotSqliteConn.Backup("main", srcSqliteConn, "main") //nolint:govet
	if err != nil {
		_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf("db snapshot: failed to init db backup - err: %+v", err) + "\n")
		return err
	}
	defer backup.Close()

	var done = false
	for !done && err == nil {
		done, err = backup.Step(250)
		if err != nil {
			_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf("db snapshot: failed to copy dbPages - err: %+v", err) + "\n")
			os.Exit(1) // in production code: bubble error and trigger retry ... instead of exiting
		}
	}
	return err
}
