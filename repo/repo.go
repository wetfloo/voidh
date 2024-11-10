package repo

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/wetfloo/voidh/file"

	_ "github.com/mattn/go-sqlite3"
)

const tableName = "fs_file"

type Repo struct {
	db              *sql.DB
	debugSelections bool
}

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

type Config struct {
	DatabasePath    string
	DebugSelections bool
}

func Init(cfg Config) (Repo, error) {
	db, err := sql.Open("sqlite3", cfg.DatabasePath)
	if err != nil {
		return Repo{}, err
	}
	if err := dbInit(db, true); err != nil {
		return Repo{}, err
	}

	repo := Repo{db: db, debugSelections: cfg.DebugSelections}
	repo.debugSelectAndPrint("init")
	return repo, nil
}

func (repo *Repo) Close() {
	repo.db.Close()
}

func (repo *Repo) Insert(file file.FsFile) error {
	if _, err := dbInteract(
		repo.db,
		fmt.Sprintf("INSERT INTO %s(fs_name, sha1) VALUES(?, ?)", tableName),
		file.Name,
		file.Hash,
	); err != nil {
		return err
	}

	repo.debugSelectAndPrint("insert")
	return nil
}

func (repo *Repo) Update(criteria Criteria, file file.FsFile) error {
	if _, err := dbInteract(
		repo.db,
		fmt.Sprintf("UPDATE %s SET fs_name = ?, sha1 = ? WHERE %s = ?", tableName, criteria.Key.dbKey()),
		file.Name,
		file.Hash,
		criteria.Value,
	); err != nil {
		return err
	}

	repo.debugSelectAndPrint("update")
	return nil
}

func (repo *Repo) Delete(criteria Criteria) error {
	if _, err := dbInteract(
		repo.db,
		fmt.Sprintf("DELETE FROM %s WHERE %s = ?", tableName, criteria.Key.dbKey()),
		criteria.Value,
	); err != nil {
		return err
	}

	repo.debugSelectAndPrint("delete")
	return nil
}

func (repo *Repo) debugSelectAndPrint(opName string) {
	var rows *sql.Rows
	var err error
	if repo.debugSelections {
		rows, err = repo.db.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
	}
	if err != nil {
		slog.Debug("can't display the result", "err", err)
	} else if rows != nil {
		for rows.Next() {
			var id int
			var fsPath string
			var sha1 string

			if err = rows.Scan(&id, &fsPath, &sha1); err != nil {
				slog.Debug("can't display the result for row", "err", err)
			}

			slog.Debug(
				"new db result",
				"opName", opName,
				"id", id,
				"fsPath", fsPath,
				"sha1", fmt.Sprintf("%x", sha1),
			)
		}

		defer rows.Close()
	}
}

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
