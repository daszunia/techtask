package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/daszunia/techtask/pkg/logs"
	"github.com/daszunia/techtask/pkg/monitor"
	"github.com/daszunia/techtask/pkg/utils"
)

const (
	helpCmd    = "help"
	exitCmd    = "exit"
	logCmd     = "log"
	viewCmd    = "view"
	filterCmd  = "filter"
	nameOpt    = "-name"
	dateOpt    = "-date"
	fromPrefix = "from="
	toPrefix   = "to="
)

var (
	doneChan    = make(chan bool, 1)
	msgChan     = make(chan string)
	logHistory  *logs.LogHistory
	fileMonitor *monitor.MonitorFiles
)

func main() {
	hotDir := flag.String("hot", "", "A path to the folder to be backed up.")
	backupDir := flag.String("backup", "", "A path to the folder where backups will be saved (optional).")
	help := flag.Bool("help", false, "Prints help message.")
	flag.Parse()

	if *help {
		utils.PrintHelp()
		return
	}

	logHistory = logs.NewLogHistory()
	fileMonitor = monitor.NewMonitorFiles(logHistory, *hotDir, *backupDir)
	fileMonitor.StartMonitoring()
	defer fileMonitor.StopMonitoring()

	waitForExit()
	readInput()

	<-doneChan
	fmt.Println("exiting")
}

func waitForExit() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			select {
			case sig := <-sigs:
				fmt.Println(sig)
				doneChan <- true
				return
			case msg := <-msgChan:
				if strings.HasPrefix(msg, helpCmd) {
					utils.PrintHelp()
				}
				if strings.HasPrefix(msg, exitCmd) {
					doneChan <- true
					return
				}
				if strings.HasPrefix(msg, logCmd) {
					handleLogView(msg)
				}
			}
		}
	}()
}

func readInput() {
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for {
			if scanner.Scan() {
				line := scanner.Text()
				msgChan <- line
			}
		}
	}()
}

func handleLogView(msg string) {
	subcommand := strings.TrimPrefix(msg, logCmd)
	subcommand = strings.TrimSpace(subcommand)
	if strings.HasPrefix(subcommand, viewCmd) {
		logHistory.PrintLog()
		return
	}

	if strings.HasPrefix(subcommand, filterCmd) {
		subcommand2 := strings.TrimPrefix(subcommand, filterCmd)
		subcommand2 = strings.TrimSpace(subcommand2)

		if strings.HasPrefix(subcommand2, nameOpt) {
			// Filter by filename regex
			filterName := strings.TrimPrefix(subcommand2, nameOpt)
			filterName = strings.TrimSpace(filterName)
			logHistory.FilterByRegex(filterName)

		} else if strings.HasPrefix(subcommand2, dateOpt) {
			// Filter by date range
			options := strings.TrimPrefix(subcommand2, dateOpt)
			options = strings.TrimSpace(options)
			optionsList := strings.Split(options, " ")

			from := time.Now().Format(utils.IsoTimeFormat)
			to := from
			for _, opt := range optionsList {
				if strings.Contains(opt, fromPrefix) {
					from = strings.TrimPrefix(opt, fromPrefix)
				}
				if strings.Contains(opt, toPrefix) {
					to = strings.TrimPrefix(opt, toPrefix)
				}
			}
			logHistory.FilterByDate(from, to)

		} else {
			fmt.Println("Unknown command")
		}
	}
}
