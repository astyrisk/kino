package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/term"
)

const (
	colorDefault = "\033[0m"
	colorRed     = "\033[91m"
	colorGreen   = "\033[92m"
	colorYellow  = "\033[93m"
	colorBlue    = "\033[94m"
	colorCyan    = "\033[96m"
)

func ClearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func getTerminalSize() (int, int) {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return 24, 80
	}
	return height, width
}

func ShowAtBottom(message string, color string) {
	rows, _ := getTerminalSize()

	newlines := rows - 1

	if newlines < 1 {
		newlines = 1
	}

	for i := 0; i < newlines; i++ {
		fmt.Println()
	}

	fmt.Print(color + message + colorDefault)
}

func ShowPromptAtBottom(prompt string) {
	ShowAtBottom(prompt, colorDefault)
}
