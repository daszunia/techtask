package utils

import (
	"fmt"
	"strings"
)

func PrintHelp() {
	fmt.Println("Usage: filefilter -hot <path> -backup <path>")
	fmt.Println("Possible commands:")
	fmt.Println("help")
	fmt.Println("exit")
	fmt.Println("logview")
	fmt.Println("logview -name <name>")
	fmt.Println("logview -date from=<date> to=<date> // date format ISO, can pass only one - from|to")
}

func GetOnlyFilename(path string) string {
	sourceSplit := strings.Split(path, "/")
	pureFilename := sourceSplit[len(sourceSplit)-1]
	return pureFilename
}
