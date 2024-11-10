package main

import (
	"crypto/sha1"
	"database/sql"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"os"

	"github.com/fsnotify/fsnotify"
	_ "github.com/mattn/go-sqlite3"
)

const tableName = "fs_file"

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug.Level())

	dir := os.Args[1]
	slog.Info("Starting to watch directory", "dir", dir)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	defer watcher.Close()

	if err := watcher.Add(dir); err != nil {
		panic(err)
	}

	db, err := sql.Open("sqlite3", "voidh.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	dbInit(db, true)

	hasher := sha1.New()

	go func() {
		for {
			select {
			case event, chanOk := <-watcher.Events:
				if !chanOk {
					slog.Debug("No more events, channel closed")
					return
				}
				fsUpdateHandle(event, hasher, db)
			case err, chanOk := <-watcher.Errors:
				if !chanOk {
					slog.Debug("No more errors, channel closed")
					return
				}
				slog.Warn("error while watching", "err", err)
			}
		}
	}()

	// Do not allow the program to quit until user request
	<-make(chan struct{})
}

func fsUpdateHandle(event fsnotify.Event, hasher hash.Hash, db *sql.DB) {
	switch {
	// TODO: for some reason this triggers twice on my (wetfloo's) machine
	case event.Has(fsnotify.Create):
		fileHash, err := fileHashCalc(event.Name, hasher)
		if err != nil {
			panic(err)
		}
		if _, err := dbInteract(db, fmt.Sprintf("INSERT INTO %s(fs_name, sha1) VALUES(?, ?)", tableName), event.Name, fileHash); err != nil {
			panic(err)
		}
		slog.Debug("fsnotify.Create", "fileName", event.Name, "fileHash", fileHash)

	// TODO: use debounce, since fsnotify.Write event doesn't mean it's done writing.
	// Maybe timeout of 2 secs is good?
	case event.Has(fsnotify.Write):
		fileHash, err := fileHashCalc(event.Name, hasher)
		if err != nil {
			panic(err)
		}
		if _, err := dbInteract(db, fmt.Sprintf("INSERT INTO %s(fs_name, sha1) VALUES(?, ?)", tableName), event.Name, fileHash); err != nil {
			panic(err)
		}
		slog.Debug("fsnotify.Create", "fileName", event.Name, "fileHash", fileHash)

	case event.Has(fsnotify.Remove):
		_, err := dbInteract(db, fmt.Sprintf("DELETE FROM %s WHERE fs_name = ?", tableName), event.Name)
		if err != nil {
			panic(err)
		}
		slog.Debug("fsnotify.Create", "fileName", event.Name)

	// TODO: some "deletion" ops may trigger fsnotify.Rename, like moving file in the trash
	// TODO: this event also means that we need to re-attach the watcher, maybe?
	case event.Has(fsnotify.Rename):
		fileHash, err := fileHashCalc(event.Name, hasher)
		if err != nil {
			panic(err)
		}
		if _, err := dbInteract(db, fmt.Sprintf("UPDATE %s SET fs_name = ? WHERE sha1 = ?", tableName), event.Name, fileHash); err != nil {
			panic(err)
		}
		slog.Debug("fsnotify.Rename", "fileName", event.Name, "fileHash", fileHash)
	}
	// other events are do not change file structure, so no need to update the db

	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		slog.Debug("can't display the result", "err", err)
	}
	if rows != nil {
		for rows.Next() {
			var id int
			var fsPath string
			var sha1 string

			if err = rows.Scan(&id, &fsPath, &sha1); err != nil {
				slog.Debug("can't display the result for row", "err", err)
			}

			slog.Debug("new db result", "id", id, "fsPath", fsPath, "sha1", sha1)
		}

		defer rows.Close()
	}
}

func dbInit(db *sql.DB, deleteIfExists bool) {
	query := fmt.Sprintf(
		`CREATE TABLE %s (
			id INTEGER NOT NULL PRIMARY KEY,
			fs_name TEXT NOT NULL,
			sha1 TEXT NOT NULL
		);
		DELETE FROM %s;`,
		tableName,
		tableName,
	)
	if deleteIfExists {
		query = fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName) + query
	}

	_, err := db.Exec(query)
	if err != nil {
		panic(err)
	}
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

func fileHashCalc(filePath string, hasher hash.Hash) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher.Reset()

	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return string(hasher.Sum(nil)), nil
}
