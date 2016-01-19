package main

import (
	"fmt"
	"github.com/howeyc/fsnotify"
	"os"
	"path/filepath"
	"regexp"
)

const DefaultWatchedFiles = "go"

type Watcher struct {
	WatchedDir  string
	Changed     bool
	Watcher     *fsnotify.Watcher
	FilePattern string
}

func NewWatcher(dir, filePattern string) (w Watcher) {
	w.WatchedDir = dir
	w.FilePattern = DefaultWatchedFiles
	if len(filePattern) != 0 {
		w.FilePattern = filePattern
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	w.Watcher = watcher

	return
}

func (this *Watcher) Watch() (err error) {
	for _, dir := range this.dirsToWatch() {
		err = this.Watcher.Watch(dir)
		if err != nil {
			return
		}
	}

	expectedFileReg := regexp.MustCompile(`\.(` + this.FilePattern + `)$`)
	for {
		file := <-this.Watcher.Event
		if expectedFileReg.Match([]byte(file.Name)) {
			fmt.Println("== Change detected:", file.Name)
			this.Changed = true
		}
	}
	return nil
}

func (this *Watcher) dirsToWatch() (dirs []string) {
	ignoredPathReg := regexp.MustCompile(`(public)|(\/\.\w+)|(^\.)|(\.\w+$)`)
	matchedDirs := make(map[string]bool)
	matchedDirs["./"] = true
	filepath.Walk(this.WatchedDir, func(filePath string, info os.FileInfo, e error) (err error) {
		if !info.IsDir() || ignoredPathReg.Match([]byte(filePath)) {
			return
		}
		if mch, _ := matchedDirs[filePath]; mch {
			return
		}

		matchedDirs[filePath] = true
		return
	})

	for dir, _ := range matchedDirs {
		dirs = append(dirs, dir)
	}
	return
}

func (this *Watcher) Reset() {
	this.Changed = false
}
