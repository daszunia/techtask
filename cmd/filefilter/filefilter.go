package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/daszunia/techtask/pkg/utils"
)

var (
	doneChan = make(chan bool, 1)
	msgChan  = make(chan string)
)

func main() {
	hotDir := flag.String("hot", "", "A path to the folder to be backed up.")
	backupDir := flag.String("backup", "", "A path to the folder where backups will be saved.")
	help := flag.Bool("help", false, "Prints help message.")
	flag.Parse()

	if *help {
		utils.PrintHelp()
		return
	}

	err := utils.ValidateFolders(*hotDir, *backupDir)
	if err != nil {
		fmt.Println("Error validating folders: ", err)
		return
	}

	fmt.Println("Monitoring files in:", *hotDir)
	fmt.Println("Saving backup to:", *backupDir)

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
				fmt.Println("Echoing: ", msg)
			}
		}
	}()
}

func readInput() {
	go func() {
		for {
			var s string
			fmt.Scanln(&s)
			msgChan <- s
		}
	}()
}
