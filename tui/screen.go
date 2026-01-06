package tui

import (
	"fmt"
	"os"
	"sync"

	"github.com/gdamore/tcell/v2"
)

// Define styles centrally
var (
	styleDefault = tcell.StyleDefault
	styleRed     = tcell.StyleDefault.Foreground(tcell.ColorRed)
	styleGreen   = tcell.StyleDefault.Foreground(tcell.ColorGreen)
	styleBlue    = tcell.StyleDefault.Foreground(tcell.ColorBlue)
	styleCyan    = tcell.StyleDefault.Foreground(tcell.ColorAqua)
)

const maxMessages = 10

type Logger struct {
	screen tcell.Screen
	msgs []string
	mu sync.Mutex
}

func NewLogger() (*Logger, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed to create screen: %w", err)
	}
	if err = s.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize screen: %w", err)
	}

	return &Logger {
		screen: s,
		msgs: make([]string, 0, maxMessages),
	}, nil

}

// Wrapper methods to satisfy tcell usage
func (l *Logger) Close()     { l.screen.Fini() }
func (l *Logger) Sync()      { l.screen.Sync() }
func (l *Logger) Clear()     { l.screen.Clear() }
func (l *Logger) PollEvent() tcell.Event { return l.screen.PollEvent() }

func (l *Logger) Size() (int, int) {
	return l.screen.Size()
}

// Core drawing logic
// TODO x+i y +i no?
func (l *Logger) drawText(x, y int, text string, style tcell.Style) {
	for i, r := range text {
		l.screen.SetContent(x+i, y, r, nil, style)
	}
}

func (l *Logger) ShowAtBottom(message string, style tcell.Style) {
	l.Clear()
	_, h := l.Size()
	l.drawText(0, h-1, message, style)
	l.screen.Show()
}

func (l *Logger) ShowWithContext(context, message string, ctxStyle, msgStyle tcell.Style) {
	l.Clear()
	if context != "" {
		l.drawText(0, 0, context, ctxStyle)
	}
	_, h := l.Size()
	l.drawText(0, h-1, message, msgStyle)
	l.screen.Show()
}

// output handles both TUI drawing and standard I/O
func (l *Logger) output(message string, style tcell.Style, isError bool) {
	// l.ShowAtBottom(message, style)

	// Also print to stdout/stderr for logging persistence
	if isError {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	} else {
		fmt.Println(message)
	}
}

// Public API
func (l *Logger) ShowInfo(msg string)    { l.output(msg, styleDefault, false) }
func (l *Logger) ShowSuccess(msg string) { l.output(msg, styleGreen, false) }
func (l *Logger) ShowStatus(msg string)  { l.output(msg, styleCyan, false) }
func (l *Logger) ShowError(msg string)   { l.output(msg, styleRed, true) }

func (l *Logger) ShowPrompt(prompt string) {
	l.ShowAtBottom(prompt, styleBlue)
	fmt.Print(prompt)
}

func (l *Logger) ShowPromptWithContext(context, prompt string) {
	l.ShowWithContext(context, prompt, styleDefault, styleBlue)
	if context != "" {
		fmt.Println(context)
	}
	fmt.Print(prompt)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.output(fmt.Sprintf(format, args...), styleDefault, false)
}

func (l *Logger) Println(args ...interface{}) {
	l.output(fmt.Sprint(args...), styleDefault, false)
}
