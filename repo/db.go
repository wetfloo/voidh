package repo

import (
	"database/sql"
	"fmt"
)

const tableName = "fs_file"

// TODO: deleteIfExists will only exist during prototyping and should never be used in prod
func dbInit(db *sql.DB, deleteIfExists bool) error {
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

	_, err := db.Exec(query)
	if err != nil {
		return err
	}

	return nil
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
