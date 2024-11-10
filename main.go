package main

import (
	"github.com/wetfloo/voidh/repo"
	"github.com/wetfloo/voidh/watch"
	"log/slog"
	"os"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug.Level())

	dir := os.Args[1]
	slog.Info("Starting to watch directory", "dir", dir)

	repo, err := repo.Init(repo.Config{
		DatabasePath:    "voidh.db",
		DebugSelections: true,
	})
	if err != nil {
		panic(err)
	}

	watch, err := watch.New(repo, dir)
	if err != nil {
		panic(err)
	}

	go watch.Start()
	defer watch.Stop()

	// Do not allow the program to quit until user request
	<-make(chan struct{})
}
