package repo

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/wetfloo/voidh/file"
)

type Repo struct {
	db              *sql.DB
	debugSelections bool
}

type Config struct {
	DatabasePath    string
	DebugSelections bool
	RemoveIfExists  bool
}

func Init(cfg Config) (Repo, error) {
	var result Repo

	db, err := dbInit(cfg.DatabasePath, cfg.RemoveIfExists)
	if err != nil {
		return result, err
	}

	result = Repo{db: db, debugSelections: cfg.DebugSelections}
	result.debugSelectAndPrint("init")
	return result, nil
}

func (repo *Repo) Close() {
	repo.db.Close()
}

func (repo *Repo) Insert(file file.FsFile) error {
	if _, err := dbInteract(
		repo.db,
		"INSERT INTO fs_file(fs_name, sha1) VALUES(?, ?)",
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
		fmt.Sprintf("UPDATE fs_file SET fs_name = ?, sha1 = ? WHERE %s = ?", criteria.Key.dbKey()),
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
		fmt.Sprintf("DELETE FROM fs_file WHERE %s = ?", criteria.Key.dbKey()),
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

	rows, err := repo.db.Query("SELECT * FROM fs_file")
	if err != nil {
		slog.Debug("can't display the result", "err", err)
		return
	}

	for rows.Next() {
		var id int
		var fsPath string
		sha1 := make([]byte, 20, 20)

		if err = rows.Scan(&id, &fsPath, &sha1); err != nil {
			slog.Debug("can't display the result for row", "err", err)
		}

		slog.Debug(
			"new db result",
			"opName", opName,
			"id", id,
			"fsPath", fsPath,
			"sha1", hex.EncodeToString(sha1),
		)
	}

	defer rows.Close()
}
