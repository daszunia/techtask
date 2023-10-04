package monitor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/daszunia/techtask/pkg/logs"
	"github.com/daszunia/techtask/pkg/utils"
)

const (
	configFilename     = ".filefilterconf"
	defaultBackupDir   = ".backup"
	hotdirConfigKey    = "hotdir="
	backupdirConfigKey = "backupdir="
	deletePrefix       = "delete_"
	swapSuffix         = ".swp"
	backupSuffix       = ".bak"
	scheduledRegex     = `delete_(\d{4}-[01]?\d-[0-3]?\dT[0-2]\d:[0-5]\d:[0-5]\d[+-]\d{4})_.*?`
	isoTimeFormat      = "2006-01-02T15:04:05-0700"
	backupOp           = "BACKUP"
	backupDelOp        = "BACKUP_DEL"
)

type MonitorFiles struct {
	logHistory *logs.LogHistory
	hotDir     string
	backupDir  string
	watcher    *fsnotify.Watcher
}

func NewMonitorFiles(logHistory *logs.LogHistory, hotDir, backupDir string) *MonitorFiles {
	mf := &MonitorFiles{logHistory: logHistory}
	ok, err := mf.verifyConfig(hotDir, backupDir)
	if !ok {
		log.Fatal(err)
	}

	fmt.Println("Monitoring files in:", mf.hotDir)
	fmt.Println("Saving backup to:", mf.backupDir)

	return mf
}

func (mf *MonitorFiles) verifyConfig(hotDir, backupDir string) (bool, error) {
	// if hotDir is empty, check if config is present
	configuredHotDir := hotDir
	configuredBackupDir := backupDir
	if hotDir == "" {
		if _, err := os.Stat(configFilename); os.IsNotExist(err) {
			log.Println("Hot dir not provided, previous config not found. Please provide hot dir.")
			return false, err
		}

		// read config file
		file, err := os.OpenFile(configFilename, os.O_RDONLY, 0644)
		if err != nil {
			log.Println("Could not open: ", configFilename)
			return false, err
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, hotdirConfigKey) {
				configuredBackupDir = strings.TrimPrefix(line, hotdirConfigKey)
			}
			if configuredBackupDir == "" && strings.HasPrefix(line, backupdirConfigKey) {
				configuredBackupDir = strings.TrimPrefix(line, backupdirConfigKey)
			}
		}
		file.Close()
	}

	if configuredBackupDir == "" {
		configuredBackupDir = defaultBackupDir
	}

	// check if hotDir exists
	if _, err := os.Stat(configuredHotDir); os.IsNotExist(err) {
		log.Println("Provided Hot dir does not exist.")
		return false, err
	}

	// check if backupDir exists, if not, create it
	if _, err := os.Stat(configuredBackupDir); os.IsNotExist(err) {
		if err := os.Mkdir(configuredBackupDir, os.ModePerm); err != nil {
			log.Println("Could not create backup dir: ", configuredBackupDir)
			return false, err
		}
	}

	// save current dirs to config file
	file, err := os.OpenFile(configFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Println("Could not open: ", configFilename)
		return false, err
	}
	configString := fmt.Sprintf("%s%s\n%s%s\n", hotdirConfigKey, configuredHotDir, backupdirConfigKey, configuredBackupDir)
	_, err = file.WriteString(configString)
	if err != nil {
		log.Println("Could not write config: ", configString)
		return false, err
	}

	mf.hotDir = configuredHotDir
	mf.backupDir = configuredBackupDir

	return true, nil
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
					log.Println("error reading event")
				}
				switch event.Op {
				case fsnotify.Create:
					fallthrough
				case fsnotify.Write:
					fallthrough
				case fsnotify.Rename:
					mf.HandlePrefix(event)
				case fsnotify.Remove:
					mf.HandleDelete(event)
				}
			case err, ok := <-mf.watcher.Errors:
				if !ok {
					log.Println("error reading error")
				}
				log.Println("error:", err)
			}
		}
	}()
}

func (mf *MonitorFiles) StopMonitoring() {
	mf.watcher.Close()
}

func (mf *MonitorFiles) HandleBackup(event fsnotify.Event) {
	timestamp := time.Now().Format(time.RFC3339)
	filepath := event.Name
	operation := event.Op.String()

	if strings.HasSuffix(filepath, swapSuffix) {
		return
	}

	mf.logHistory.AddToHistory(timestamp, filepath, operation)
	err := mf.copyFile(filepath)
	if err != nil {
		log.Println("error:", err)
	}
}

func (mf *MonitorFiles) HandlePrefix(event fsnotify.Event) {
	filepath := event.Name
	pureFilename := utils.GetOnlyFilename(filepath)
	if !strings.HasPrefix(pureFilename, deletePrefix) {
		mf.HandleBackup(event)
		return
	}

	// check for scheduled delete
	r := regexp.MustCompile(scheduledRegex)
	matches := r.FindStringSubmatch(pureFilename)
	if len(matches) > 1 {
		scheduleTime, err := time.Parse(isoTimeFormat, matches[1])
		if err != nil {
			log.Println("Error parsing date: ", err)
			return
		}
		if scheduleTime.Before(time.Now()) {
			err := mf.deleteFile(filepath, false, "")
			if err != nil {
				log.Println("error:", err)
			}
			return
		}

		// schedule delete
		go func() {
			mf.waitUntil(context.Background(), scheduleTime)
			log.Println("Deleting scheduled file: ", filepath)
			err := mf.deleteFile(filepath, true, matches[1])
			if err != nil {
				log.Println("error:", err)
			}
		}()
	} else {
		err := mf.deleteFile(filepath, false, "")
		if err != nil {
			log.Println("error:", err)
		}
	}
}

func (mf *MonitorFiles) HandleDelete(event fsnotify.Event) {
	timestamp := time.Now().Format(time.RFC3339)
	filepath := event.Name
	operation := event.Op.String()
	// log removing original file
	mf.logHistory.AddToHistory(timestamp, filepath, operation)
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

	destFile := mf.backupFilename(sourceFile)
	destination, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)

	mf.logHistory.AddToHistory(time.Now().Format(time.RFC3339), destFile, backupOp)
	return err
}

func (mf *MonitorFiles) deleteFile(sourceFile string, withTimestamp bool, isotimestamp string) error {
	// remove original file
	err := os.Remove(sourceFile)
	if err != nil {
		return err
	}

	// remove backup file
	noprefixName := strings.TrimPrefix(utils.GetOnlyFilename(sourceFile), deletePrefix)
	if withTimestamp {
		noprefixName = strings.TrimPrefix(noprefixName, isotimestamp+"_")
	}
	backupFileName := mf.backupFilename(noprefixName)
	err = os.Remove(backupFileName)
	if err != nil {
		return err
	}

	// log deleting backup
	mf.logHistory.AddToHistory(time.Now().Format(time.RFC3339), backupFileName, backupDelOp)
	return nil
}

func (mf *MonitorFiles) waitUntil(ctx context.Context, until time.Time) {
	timer := time.NewTimer(time.Until(until))
	defer timer.Stop()

	select {
	case <-timer.C:
		return
	case <-ctx.Done():
		return
	}
}

func (mf *MonitorFiles) backupFilename(sourceFile string) string {
	newFilename := utils.GetOnlyFilename(sourceFile) + backupSuffix
	destFile := filepath.Join(mf.backupDir, newFilename)
	return destFile
}
