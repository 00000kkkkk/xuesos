package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"

	xuesos "github.com/00000kkkkk/xusesosplusplus/cmd/xuesos"
)

func main() {
	if err := xuesos.Execute(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		waitIfDoubleClicked()
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		waitIfDoubleClicked()
	}
}

// waitIfDoubleClicked pauses on Windows when launched without a terminal (double-click).
func waitIfDoubleClicked() {
	if runtime.GOOS == "windows" {
		fmt.Print("\nPress Enter to close...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}
}
