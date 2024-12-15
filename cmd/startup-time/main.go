package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const stateFile = "/var/lib/startup-time.state"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: startup-time <set|get> [time]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "set":
		if len(os.Args) < 3 {
			fmt.Println("Usage: startup-time set <time>")
			os.Exit(1)
		}
		inputTime := strings.Join(os.Args[2:], " ")
		parsedTime, err := parseTime(inputTime)
		if err != nil {
			fmt.Println("Error parsing time:", err)
			os.Exit(1)
		}
		writeTime(parsedTime)
	case "get":
		readTime()
	default:
		fmt.Println("Unknown command:", os.Args[1])
		os.Exit(1)
	}
}

func parseTime(input string) (string, error) {
	cmd := exec.Command("date", "--date="+input, "--iso-8601=seconds")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to parse time using date: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func writeTime(newTime string) {
	file, err := os.OpenFile(stateFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer file.Close()

	_, err = file.WriteString(newTime)
	if err != nil {
		fmt.Println("Failed to write time:", err)
		os.Exit(1)
	}
	fmt.Println("Set startup time to", newTime)
}

func readTime() {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		fmt.Println("Failed to read state file:", err)
		os.Exit(1)
	}
	fmt.Println("Startup time is", string(data))
}
