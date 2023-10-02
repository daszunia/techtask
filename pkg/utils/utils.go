package utils

import "fmt"

func ValidateFolders(hotDir, backupDir string) error {
	if hotDir == "" || backupDir == "" {
		PrintHelp()
		return fmt.Errorf("hot and backup directories must be provided")
	}

	return nil
}

func PrintHelp() {
	fmt.Println("Usage: filefilter -hot <path> -backup <path>")
	fmt.Println("Possible commands:")
}
