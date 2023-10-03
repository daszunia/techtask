package monitor

import (
	"log"

	"github.com/fsnotify/fsnotify"

	"github.com/daszunia/techtask/pkg/logs"
)

type MonitorFiles struct {
	logHistory *logs.LogHistory
	hotDir     string
	backupDir  string
	watcher    *fsnotify.Watcher
}

func NewMonitorFiles(logHistory *logs.LogHistory, hotDir, backupDir string) *MonitorFiles {
	return &MonitorFiles{
		logHistory: logHistory,
		hotDir:     hotDir,
		backupDir:  backupDir,
	}
}

func (mf *MonitorFiles) StartMonitoring() {
	var err error
	mf.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer mf.watcher.Close()

	// Start listening for events.
	go func() {
		for {
			select {
			case event, ok := <-mf.watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Has(fsnotify.Write) {
					log.Println("modified file:", event.Name)
				}
			case err, ok := <-mf.watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	// Add a path.
	err = mf.watcher.Add(mf.hotDir)
	if err != nil {
		log.Fatal(err)
	}
}
