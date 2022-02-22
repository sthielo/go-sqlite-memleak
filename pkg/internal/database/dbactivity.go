package database

import (
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"gopkg.in/errgo.v2/errors"
	"io"
	"os"
	"time"
)

const ActivityDump = "DUMP"
const ActivityNone = "NONE"
const ActivityOther = "OTHER"

var invalidDbActivityCmd = errors.New("invalid db activity command - expected: DUMP, OTHER, NONE")

/*
 * as dumping a db to sql stmts is a fairly slow process, the export first makes an in-memory backup (snapshot) of the
 * database, which can then be dumped without blocking the main db for regular usage (e.g. UI requests)
 * as snapshotting is non-invasive, meaning it offers time slots for requests to happen, it may fail when such requests
 * update/change the database => export could be retried ... done so in our productive project
 */
func Activity(cmd string) error {
	_, _ = os.Stdout.WriteString(">>> oom: " + ("starting export ...\n") + "\n")

	// "snapshotting from in-memory db to another in-memory db (using distinct file urls) seems to be the root trigger for the observed memory leak
	err := withSnapshotDo(func(dbToBackup *sql.DB) error {
		if cmd == ActivityDump {
			// original code to observe described memoey leak - intense db activity seems to make the memory leak more "obvious"
			// => almost every iteration shows a memory growth
			return dumpToFile(dbToBackup) // snapshot db activity

		} else if cmd == ActivityNone {
			// snapshot only without any activity on that snapshot
			// => only shows a memory growth the first few iterations and then only occasionally (like a log curve)

		} else {
			return invalidDbActivityCmd
		}

		return nil
	})

	// VERIFICATION check: dump from main db, so NOT using "snapshotting" => no memory leak!
	//err := dumpToFile(MyDb)

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

	// ORIG using github.com/schollz/sqlite3dump to dump db
	// err = sqlite3dump.DumpDB(dbToBackup, gw, sqlite3dump.WithMigration())
	err = alternativeDump(dbToBackup, gw)
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

	// ORIG:
	snapshotConnStr := fmt.Sprintf("file:%s?mode=memory&cache=private&_journal_mode=OFF&_fk=off&_query_only=true&_locking=EXCLUSIVE&_mutex=no", file.Name())

	// TESTING some conn str uri params => no effect - still memory leaking
	// snapshotConnStr := fmt.Sprintf("file:%s?mode=memory&cache=private&_journal_mode=OFF&_fk=off&_mutex=no", file.Name())

	// WORKAROUND: using file based db instead of in-mem (mode=rwc) => slower when creating snapshot db:
	//snapshotConnStr := fmt.Sprintf("file:%s?mode=rwc&cache=private&_journal_mode=OFF&_fk=off&_query_only=true&_locking=EXCLUSIVE&_mutex=no", file.Name())

	// >>> proposed in https://github.com/mattn/go-sqlite3/issues/1005#issuecomment-1019029882 : use `:memory:`instead of temp file name
	//     => no impact on increasing memory consumption behavior
	//snapshotConnStr := "file::memory:?mode=memory&cache=private&_journal_mode=OFF&_fk=off&_query_only=true&_locking=EXCLUSIVE&_mutex=no"

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

const stmtTables = `SELECT "name", "type", "sql" 
		FROM "sqlite_master" 
		WHERE "sql" NOT NULL AND "type" == 'table' 
		ORDER BY "name"`

type ColumnInfo struct {
	colName string
	colType string
}
type TableInfo struct {
	columnInfos []*ColumnInfo
}

/**
 * simplified alternative implementation not to depend on github.com/schollz/sqlite3dump
 */
func alternativeDump(db *sql.DB, file io.Writer) error {
	_, err := file.Write([]byte("BEGIN TRANSACTION;\n"))
	failOnErr("write tx begin", err)

	tableNames := getTableNames(db)

	for _, tableName := range tableNames {
		tableInfo := getTableInfo(db, tableName)

		stmtpartColNames := ""
		stmtpartColValues := ""
		for i, ci := range tableInfo.columnInfos {
			if i > 0 {
				stmtpartColNames += ", "
				stmtpartColValues += ", "
			}
			stmtpartColNames += "\"" + ci.colName + "\""
			if ci.colType == "text" {
				stmtpartColValues += "' || quote(\"" + ci.colName + "\") || '"
			} else {
				stmtpartColValues += "' || \"" + ci.colName + "\" || '"
			}
		}

		stmtInsStmts := "SELECT 'INSERT INTO \"" + tableName + "\"(" + stmtpartColNames + ")" +
			" VALUES(" + stmtpartColValues + ")' from \"" + tableName + "\""
		dumpInsStmts(db, stmtInsStmts, file)
	}

	_, err = file.Write([]byte("COMMIT;\n"))
	failOnErr("write commit", err)

	return nil
}

func getTableNames(db *sql.DB) []string {
	tableRows, err := db.Query(stmtTables)
	failOnErr("query tables", err)
	defer tableRows.Close()

	tableNames := make([]string, 0, 10)
	for tableRows != nil && tableRows.Next() {
		var tableName string
		var tableType string
		var creationStmt string
		err = tableRows.Scan(&tableName, &tableType, &creationStmt)
		failOnErr("step tables", err)
		tableNames = append(tableNames, tableName)
	}
	return tableNames
}

func getTableInfo(db *sql.DB, tableName string) *TableInfo {
	stmtTableInfo := "PRAGMA table_info('" + tableName + "')"
	rs, err := db.Query(stmtTableInfo)
	failOnErr("table info", err)
	defer rs.Close()

	var colInfos = make([]*ColumnInfo, 0, 3)
	var colId int
	var colName string
	var colType string
	var nullable int
	var defaultVal sql.NullString
	var pk int
	for rs != nil && rs.Next() {
		err = rs.Scan(&colId, &colName, &colType, &nullable, &defaultVal, &pk)
		failOnErr("parse table info", err)
		colInfos = append(colInfos, &ColumnInfo{
			colName: colName,
			colType: colType,
		})
	}

	return &TableInfo{columnInfos: colInfos}
}

func dumpInsStmts(db *sql.DB, stmtInsStmts string, file io.Writer) {
	insRows, err := db.Query(stmtInsStmts)
	failOnErr("query table content", err)
	defer insRows.Close()

	for insRows != nil && insRows.Next() {
		var insStmt string
		err = insRows.Scan(&insStmt)
		failOnErr("step insStmts", err)

		_, err = file.Write([]byte(insStmt + ";\n"))
		failOnErr("write insStmts", err)
	}

}
