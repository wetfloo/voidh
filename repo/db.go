package repo

import (
	"database/sql"

	_ "embed"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed sql/fs_file/init.sql
var initQuery string

//go:embed sql/fs_file/drop_existing.sql
var dropExistingQuery string

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

	if deleteIfExists {
		if _, err := db.Exec(dropExistingQuery); err != nil {
			return db, err
		}
	}

	if _, err := db.Exec(initQuery); err != nil {
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
