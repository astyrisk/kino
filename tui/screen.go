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

const maxMessages = 20

type Logger struct {
	screen tcell.Screen
	msgs   []string
	mu     sync.Mutex
}

func NewLogger() (*Logger, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed to create screen: %w", err)
	}
	if err = s.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize screen: %w", err)
	}

	return &Logger{
		screen: s,
		msgs:   make([]string, 0, maxMessages),
	}, nil
}

// Wrapper methods to satisfy tcell usage
func (l *Logger) Close()                 { l.screen.Fini() }
func (l *Logger) Sync()                  { l.screen.Sync() }
func (l *Logger) Clear()                 { l.screen.Clear() }
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

// Public API
func (l *Logger) ShowInfo(msg string)    { l.PrintAtBottom(msg) }
func (l *Logger) ShowSuccess(msg string) { l.PrintAtBottom(msg) }
func (l *Logger) ShowStatus(msg string)  { l.PrintAtBottom(msg) }
func (l *Logger) ShowError(msg string)   { l.PrintAtBottom(msg) }

func (l *Logger) Suspend() {
	l.screen.Fini()
}

func (l *Logger) Resume() error {
	return l.screen.Init()
}

// WaitForEnter waits for the user to press Enter
func (l *Logger) WaitForEnter() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for {
		ev := l.screen.PollEvent()
		if keyEv, ok := ev.(*tcell.EventKey); ok {
			if keyEv.Key() == tcell.KeyEscape {
				os.Exit(0)
			} else if keyEv.Key() == tcell.KeyEnter {
				return
			}
		}
	}
}

// write a simple function to print at the bottom from scratch
func (l *Logger) PrintAtBottom(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Add message to stack
	l.msgs = append(l.msgs, message)
	if len(l.msgs) > maxMessages {
		l.msgs = l.msgs[1:]
	}

	l.Clear()
	_, h := l.Size()

	// Draw messages from oldest to newest (bottom)
	startY := h - len(l.msgs)
	for i, msg := range l.msgs {
		l.drawText(0, startY+i, msg, styleDefault)
	}

	l.screen.Show()
}

func (l *Logger) PromptAtBottom(prompt string) string {
	l.mu.Lock()
	defer l.mu.Unlock()

	input := ""

	for {
		// clean the stack
		l.msgs = l.msgs[:0]

		// Add prompt with current input to stack
		l.msgs = append(l.msgs, prompt+input)

		l.Clear()
		_, h := l.Size()

		// Draw messages from oldest to newest (bottom)
		startY := h - len(l.msgs)
		for i, msg := range l.msgs {
			l.drawText(0, startY+i, msg, styleDefault)
		}

		l.screen.Show()

		// Wait for user input
		ev := l.screen.PollEvent()
		if keyEv, ok := ev.(*tcell.EventKey); ok {
			if keyEv.Key() == tcell.KeyEscape {
				os.Exit(0)
			} else if keyEv.Key() == tcell.KeyEnter {
				l.msgs = append(l.msgs, input)
				return input
			} else if keyEv.Key() == tcell.KeyBackspace || keyEv.Key() == tcell.KeyBackspace2 {
				if len(input) > 0 {
					input = input[:len(input)-1]
				}
			} else if keyEv.Rune() != 0 {
				input += string(keyEv.Rune())
			}
		}
	}
}
