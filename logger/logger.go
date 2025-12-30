package logger

import (
	"fmt"
	"os"
)

type Logger struct {
}

func New() *Logger {
	return &Logger{}
}

func (l *Logger) ClearScreen() {
	ClearScreen()
}

func (l *Logger) ShowPrompt(message string) {
	l.ClearScreen()
	showAtBottom(message, colorBlue)
}

func (l *Logger) ShowInfo(message string) {
	l.ClearScreen()
	showAtBottom(message, colorDefault)
}

func (l *Logger) ShowError(message string) {
	l.ClearScreen()
	showAtBottom("Error: "+message, colorRed)
}

func (l *Logger) ShowSuccess(message string) {
	l.ClearScreen()
	showAtBottom(message, colorGreen)
}

func (l *Logger) ShowMessage(message string) {
	showAtBottom(message, colorDefault)
}

func (l *Logger) ShowPromptWithContext(context, prompt string) {
	l.ClearScreen()
	if context != "" {
		fmt.Println(context)
		fmt.Println()
	}
	showAtBottom(prompt, colorBlue)
}

func (l *Logger) ShowStatus(message string) {
	l.ClearScreen()
	showAtBottom(message, colorCyan)
}

func (l *Logger) Error(message string) {
	l.ShowError(message)
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Print(message)
}

func (l *Logger) Println(args ...interface{}) {
	fmt.Println(args...)
}
