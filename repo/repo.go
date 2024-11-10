package repo

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/wetfloo/voidh/file"
)

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
	RemoveIfExists  bool
}

func Init(cfg Config) (Repo, error) {
	db, err := sql.Open("sqlite3", cfg.DatabasePath)
	if err != nil {
		return Repo{}, err
	}
	if err := dbInit(db, cfg.RemoveIfExists); err != nil {
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
	if !repo.debugSelections {
		return
	}

	rows, err := repo.db.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		slog.Debug("can't display the result", "err", err)
		return
	}

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
