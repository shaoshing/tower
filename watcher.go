package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/howeyc/fsnotify"
)

const DefaultWatchedFiles = "go"

var (
	eventTime    = make(map[string]int64)
	scheduleTime time.Time
)

type Watcher struct {
	WatchedDir  string
	Changed     bool
	OnChanged   func(string)
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
		// Skip TMP files for Sublime Text.
		if checkTMPFile(file.Name) {
			continue
		}
		if expectedFileReg.Match([]byte(file.Name)) {
			mt := getFileModTime(file.Name)
			if t := eventTime[file.Name]; mt == t {
				fmt.Printf("[SKIP] # %s #\n", file.String())
				eventTime[file.Name] = mt
				continue
			}
			eventTime[file.Name] = mt
			fmt.Println("== Change detected:", file.Name)
			this.Changed = true
			if this.OnChanged != nil {
				go func() {
					// Wait 1s before autobuild util there is no file change.
					scheduleTime = time.Now().Add(1 * time.Second)
					for {
						time.Sleep(scheduleTime.Sub(time.Now()))
						if time.Now().After(scheduleTime) {
							break
						}
						return
					}
					this.OnChanged(file.Name)
				}()
			}
		}
	}
	return nil
}

func (this *Watcher) dirsToWatch() (dirs []string) {
	ignoredPathReg := regexp.MustCompile(`(public)|(\/\.\w+)|(^\.)|(\.\w+$)`)
	matchedDirs := make(map[string]bool)
	matchedDirs["./"] = true
	filepath.Walk(this.WatchedDir, func(filePath string, info os.FileInfo, e error) (err error) {
		if e != nil {
			return e
		}
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

// checkTMPFile returns true if the event was for TMP files.
func checkTMPFile(name string) bool {
	if strings.HasSuffix(strings.ToLower(name), ".tmp") {
		return true
	}
	return false
}

// getFileModTime retuens unix timestamp of `os.File.ModTime` by given path.
func getFileModTime(path string) int64 {
	path = strings.Replace(path, "\\", "/", -1)
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("[ERRO] Fail to open file[ %s ]\n", err)
		return time.Now().Unix()
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		fmt.Printf("[ERRO] Fail to get file information[ %s ]\n", err)
		return time.Now().Unix()
	}

	return fi.ModTime().Unix()
}
