package logs

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/daszunia/techtask/pkg/utils"
)

const (
	logdir      = ".logs"
	logfileName = "log.txt"
)

type LogHistory struct {
	mu          sync.Mutex
	logfilePath string
}

func NewLogHistory() *LogHistory {
	lh := &LogHistory{}

	if _, err := os.Stat(logdir); os.IsNotExist(err) {
		if err := os.Mkdir(logdir, os.ModePerm); err != nil {
			log.Fatal(err)
		}
	}
	lh.logfilePath = filepath.Join(logdir, logfileName)
	return lh
}

func (lh *LogHistory) AddToHistory(filename, operation string) {
	lh.mu.Lock()
	defer lh.mu.Unlock()
	file, err := os.OpenFile(lh.logfilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Println("Could not open: ", lh.logfilePath)
		return
	}

	defer file.Close()
	timestamp := time.Now().Format(utils.IsoTimeFormat)
	logline := fmt.Sprintf("%s %s %s\n", timestamp, filename, operation)
	_, err = file.WriteString(logline)
	if err != nil {
		log.Println("Could not write log: ", logline)
	}
}

func (lh *LogHistory) PrintLog() {
	b, err := os.ReadFile(lh.logfilePath)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("************************************************************")
	fmt.Print(string(b))
	fmt.Println("************************************************************")
}

func (lh *LogHistory) FilterByDate(fromDate, toDate string) {
	from := fmt.Sprintf("from=%s", fromDate)
	to := fmt.Sprintf("to=%s", toDate)

	args := []string{"-v", from, "-v", to, "{ if ($1 >= from && $1 <= to) { print $0 } }", lh.logfilePath}
	out, err := exec.Command("awk", args...).Output()
	if err != nil {
		log.Println("Error filtering log file: ", err)
		return
	}

	fmt.Println("************************************************************")
	fmt.Print(string(out))
	fmt.Println("************************************************************")
}

func (lh *LogHistory) FilterByRegex(filenameRegex string) {
	regex := fmt.Sprintf("$2~/%s/", filenameRegex)
	args := []string{regex, lh.logfilePath}
	out, err := exec.Command("awk", args...).Output()
	if err != nil {
		log.Println("Error filtering log file: ", err)
		return
	}

	fmt.Println("************************************************************")
	fmt.Print(string(out))
	fmt.Println("************************************************************")
}
