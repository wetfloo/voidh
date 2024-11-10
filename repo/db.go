package repo

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

const tableName = "fs_file"

type Criteria struct {
	Key   Key
	Value any
}

type Key interface {
	dbKey() string
}

type Filename struct{}

func (_ Filename) dbKey() string {
	return "fs_name"
}

type Hash struct{}

func (_ Hash) dbKey() string {
	return "sha1"
}

// TODO: deleteIfExists will only exist during prototyping and should never be used in prod
func dbInit(databasePath string, deleteIfExists bool) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		return db, err
	}

	query := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s (
			id INTEGER NOT NULL PRIMARY KEY,
			fs_name TEXT NOT NULL,
			sha1 BLOB NOT NULL
		) STRICT;`,
		tableName,
	)
	if deleteIfExists {
		query = fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName) + query
	}

	if _, err := db.Exec(query); err != nil {
		return db, err
	}

	return db, nil
}

func dbInteract(db *sql.DB, query string, args ...any) (sql.Result, error) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	result, err := stmt.Exec(args...)
	if err != nil {
		return result, err
	}

	err = tx.Commit()

	return result, err
}
