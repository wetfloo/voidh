package watch

import (
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"os"

	"github.com/fsnotify/fsnotify"
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
		slog.Debug("fsnotify.Create", "fileName", event.Name, "fileHash", fmt.Sprintf("%x", fileHash))

	// TODO: use debounce, since fsnotify.Write event doesn't mean it's done writing.
	// Maybe timeout of 2 secs is good?
	case event.Has(fsnotify.Write):
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
		slog.Debug("fsnotify.Write", "fileName", event.Name, "fileHash", fmt.Sprintf("%x", fileHash))

	case event.Has(fsnotify.Remove):
		if err := watch.repo.Delete(repo.Criteria{
			Key:   repo.Filename{},
			Value: event.Name,
		}); err != nil {
			panic(err)
		}
		slog.Debug("fsnotify.Remove", "fileName", event.Name)

	// TODO: some "deletion" ops may trigger fsnotify.Rename, like moving file in the trash
	// TODO: this event also means that we need to re-attach the watcher, maybe?
	case event.Has(fsnotify.Rename):
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
		slog.Debug("fsnotify.Rename", "fileName", event.Name, "fileHash", fmt.Sprintf("%x", fileHash))
	}
	// other events are do not change file structure, so no need to update the db

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
