package database

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"regexp"
)

var MyDb *sql.DB

/**
 * creates an empty db and applies the schema
 * NOTE: all initDb code takes any error as fatal
 */
func InitDB() {
	file, err := os.CreateTemp("tmp", ".oom-*.db")
	if err != nil {
		_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf("cannot create temporary db file - err: %+v", err) + "\n")
		os.Exit(1)
	}
	_ = file.Close()

	connStr := fmt.Sprintf("file:%s?mode=memory&cache=private&_fk=1&_journal_mode=OFF&_locking=EXCLUSIVE&_mutex=no", file.Name())
	MyDb, err = sql.Open("sqlite3", connStr)
	if err != nil {
		_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf("cannot create db - err: %+v", err) + "\n")
		os.Exit(1)
	}
	MyDb.SetMaxOpenConns(10)
	createSchema(MyDb)
}

func createSchema(db *sql.DB) {
	execStmtFatalOnErr(db, `create table t1 (
									id text not null primary key CHECK (id like '________-____-____-____-____________'),
									t1f1 text,
									t1f2 integer not null check(t1f2 in(0, 1)),
									t1f3 text not null default (date('now')) CHECK (t1f3 like '____-__-__')
								) without rowid`)

	execStmtFatalOnErr(db, `create table t2 (
									id text not null CHECK (id like '________-____-____-____-____________'),
									t2f1 text not null,
									t2f2 text not null primary key,
									foreign key(id) references t1(id) on delete cascade deferrable initially deferred
								) without rowid`)

	execStmtFatalOnErr(db, `create table t3 (
									id text not null primary key CHECK (id like '________-____-____-____-____________'),
									t3f1 text,
									t3f2 text,
									t3f3 text CHECK (t3f3 like '____-__-__'),
									t3f4 integer,
									t3f5 real,
									t3f8 text,
									foreign key(id) references t1(id) on delete cascade deferrable initially deferred
								) without rowid`)

	execStmtFatalOnErr(db, `create table t5 (
									id text not null primary key CHECK (id like '________-____-____-____-____________'),
                  					t1_id text not null,
									t5f1 integer not null check(t5f1 in(0, 1)),
									t5f2 text not null CHECK (t5f2 like '____-__-__'),
									t5f3 text,
									t5f4 text not null check(t5f4 in('valA', 'valB', 'valC')),
									foreign key(t1_id) references t1(id) on delete cascade deferrable initially deferred,
									unique(t1_id, t5f4, t5f3) 
								) without rowid`)

	execStmtFatalOnErr(db, `create table t6 (
									t5_id text not null CHECK (t5_id like '________-____-____-____-____________'),
									t6f1 text not null CHECK (t6f1 like '____-__-__'),
									t6f2 real,
									primary key(t5_id, t6f1),
									foreign key(t5_id) references t5(id) on delete cascade deferrable initially deferred
								) without rowid`)

	execStmtFatalOnErr(db, `create table t10 (
									id text not null CHECK (id like '________-____-____-____-____________'),
									t10f1 text not null,
									t10f2 text not null,
									primary key(id),
									unique (t10f1, t10f2)
								) without rowid`)

	execStmtFatalOnErr(db, `create table t11 (
									t10_id text not null CHECK (t10_id like '________-____-____-____-____________'),
									t11f1 text not null CHECK (t11f1 like '____-__-__'),
									t11f2 real not null,
									primary key(t10_id, t11f1),
									foreign key(t10_id) references t10(id) on delete cascade deferrable initially deferred
								) without rowid`)
}

func execStmtFatalOnErr(db *sql.DB, stmt string) {
	commentMatcher := regexp.MustCompile("(?m) *(?:--.*)?$")
	tabNewLineMatcher := regexp.MustCompile("(?m)[\t\n]")
	strippedStmt := commentMatcher.ReplaceAllString(stmt, "")
	strippedStmt = tabNewLineMatcher.ReplaceAllString(strippedStmt, " ")
	_, err := db.Exec(strippedStmt)
	if err != nil {
		_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf("failed db schema generation: stmt=%s - err: %+v", strippedStmt, err) + "\n")
		os.Exit(1)
	}
}
