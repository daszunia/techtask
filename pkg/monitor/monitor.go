package monitor

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/daszunia/techtask/pkg/logs"
)

const (
	deletePrefix = "delete_"
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

	err = mf.watcher.Add(mf.hotDir)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case event, ok := <-mf.watcher.Events:
				if !ok {
					fmt.Println("error reading event")
				}
				switch event.Op {
				case fsnotify.Create:
					fallthrough
				case fsnotify.Write:
					mf.BackupFile(event)
				case fsnotify.Rename:
					mf.HandleRename(event)
				}
			case err, ok := <-mf.watcher.Errors:
				if !ok {
					fmt.Println("error reading error")
				}
				log.Println("error:", err)
			}
		}
	}()
}

func (mf *MonitorFiles) StopMonitoring() {
	mf.watcher.Close()
}

func (mf *MonitorFiles) BackupFile(event fsnotify.Event) {
	timestamp := time.Now().Format(time.RFC3339)
	filename := event.Name
	operation := event.Op.String()

	if strings.HasSuffix(filename, ".swp") {
		return
	}

	mf.logHistory.AddToHistory(timestamp, filename, operation)
	mf.copyFile(filename)
}

func (mf *MonitorFiles) HandleRename(event fsnotify.Event) {
	fmt.Println("Handle rename")
	log.Println(event)
}

func (mf *MonitorFiles) copyFile(sourceFile string) error {
	sourceFileStat, err := os.Stat(sourceFile)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", sourceFile)
	}

	source, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer source.Close()

	sourceSplit := strings.Split(sourceFile, "/")
	newFilename := sourceSplit[len(sourceSplit)-1] + ".bak"
	destFile := filepath.Join(mf.backupDir, newFilename)
	destination, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
