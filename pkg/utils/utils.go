package utils

import (
	"fmt"
	"strings"
)

const IsoTimeFormat = "2006-01-02T15:04:05-0700"

func PrintHelp() {
	fmt.Println("Usage: filefilter -hot <path> -backup <path>")
	fmt.Println("Possible commands:")
	fmt.Println("help")
	fmt.Println("exit")
	fmt.Println("log view")
	fmt.Println("log filter -name <name_regex>")
	fmt.Println("log filter -date from=<date> to=<date> // date format ISO")
	fmt.Println("    for example: log filter -date from=2023-10-05T00:06:02+0200 to=2023-10-05T00:06:05+0200")
	fmt.Println("")
}

func GetOnlyFilename(path string) string {
	sourceSplit := strings.Split(path, "/")
	pureFilename := sourceSplit[len(sourceSplit)-1]
	return pureFilename
}
