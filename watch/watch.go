package watch

import (
	"crypto/sha1"
	"encoding/hex"
	"hash"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	debounce "github.com/wetfloo/go_debounce"
	"github.com/wetfloo/voidh/file"
	"github.com/wetfloo/voidh/repo"
)

type Watch struct {
	watcher *fsnotify.Watcher
	repo    repo.Repo
	hasher  hash.Hash
}

func New(repo repo.Repo, dir string) (Watch, error) {
	var result Watch

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return result, err
	}

	if err := watcher.Add(dir); err != nil {
		return result, err
	}

	hasher := sha1.New()

	result = Watch{watcher, repo, hasher}
	return result, nil
}

func (watch *Watch) Start() error {
	for {
		select {
		case event, chanOk := <-watch.watcher.Events:
			if !chanOk {
				slog.Debug("No more events, channel closed")
				return nil
			}
			watch.fsUpdateHandle(event)
		case err, chanOk := <-watch.watcher.Errors:
			if !chanOk {
				slog.Debug("No more errors, channel closed")
				return nil
			}
			slog.Warn("error while watching", "err", err)
		}
	}
}

func (watch *Watch) Stop() {
	watch.watcher.Close()
}

func (watch *Watch) fsUpdateHandle(event fsnotify.Event) {
	switch {
	case event.Has(fsnotify.Create):
		fileHash, err := fileHashCalc(event.Name, watch.hasher)
		if err != nil {
			panic(err)
		}
		if err := watch.repo.Insert(file.FsFile{
			Name: event.Name,
			Hash: fileHash,
		}); err != nil {
			panic(err)
		}
		slog.Debug("fsnotify.Create", "fileName", event.Name, "fileHash", hex.EncodeToString(fileHash))

	case event.Has(fsnotify.Write):
		debounce.New(2 * time.Second)(func() {
			fileHash, err := fileHashCalc(event.Name, watch.hasher)
			if err != nil {
				panic(err)
			}
			if err := watch.repo.Update(repo.Criteria{
				Key:   repo.Filename{},
				Value: fileHash,
			}, file.FsFile{
				Name: event.Name,
				Hash: fileHash,
			}); err != nil {
				panic(err)
			}
			slog.Debug("fsnotify.Write", "fileName", event.Name, "fileHash", hex.EncodeToString(fileHash))
		})

	case event.Has(fsnotify.Remove):
		if err := watch.fsFileForget(event.Name); err != nil {
			panic(err)
		}
		slog.Debug("fsnotify.Remove", "fileName", event.Name)

	// TODO: this event also means that we need to re-attach the watcher, maybe?
	case event.Has(fsnotify.Rename):
		// File is no longer on our path, forget about it (could happen if moved to a whole new location, for example)
		// TODO: allow to watch for more than one path, this assumes there's only one
		// TODO: does this expand to absolute path? If not, we should always do that
		if hasPrefixEvenWithSurround(event.Name, "\"", watch.watcher.WatchList()[0]) {
			if err := watch.fsFileForget(event.Name); err != nil {
				panic(err)
			}
			return
		}

		fileHash, err := fileHashCalc(event.Name, watch.hasher)
		if err != nil {
			panic(err)
		}
		if err := watch.repo.Update(repo.Criteria{
			Key:   repo.Hash{},
			Value: fileHash,
		}, file.FsFile{
			Name: event.Name,
			Hash: fileHash,
		}); err != nil {
			panic(err)
		}
		slog.Debug("fsnotify.Rename", "fileName", event.Name, "fileHash", hex.EncodeToString(fileHash))
	}
	// other events are do not change file structure, so no need to update the db
}

func (watch *Watch) fsFileForget(name string) error {
	if err := watch.repo.Delete(repo.Criteria{
		Key:   repo.Filename{},
		Value: name,
	}); err != nil {
		return err
	}

	return nil
}

func fileHashCalc(filePath string, hasher hash.Hash) ([]byte, error) {
	var result []byte

	file, err := os.Open(filePath)
	if err != nil {
		return result, err
	}
	defer file.Close()

	hasher.Reset()

	if _, err := io.Copy(hasher, file); err != nil {
		return result, err
	}

	result = hasher.Sum(nil)
	return result, nil
}

// Checks if a given string has prefix, if not, tries to surround it
// with a given string and tries again
func hasPrefixEvenWithSurround(s string, sur string, prefix string) bool {
	if strings.HasPrefix(s, prefix) {
		return true
	}
	var builder strings.Builder
	builder.WriteString(sur)
	builder.WriteString(s)
	builder.WriteString(sur)
	return strings.HasPrefix(builder.String(), prefix)
}
