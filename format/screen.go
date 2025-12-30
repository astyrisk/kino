package format

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"unsafe"
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

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func getTerminalSize() (int, int) {
	ws := &winsize{}
	retCode, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		return 24, 80
	}
	return int(ws.Row), int(ws.Col)
}

func ShowPromptAtBottom(prompt string) {
	rows, _ := getTerminalSize()
	
	newlines := rows - 1 
	
	if newlines < 1 {
		newlines = 1
	}
	
	for i := 0; i < newlines; i++ {
		fmt.Println()
	}
	
	fmt.Print(prompt)
}