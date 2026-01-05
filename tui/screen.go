package tui

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
)

type Screen struct {
	screen tcell.Screen
}

var (
	ColorDefault = tcell.StyleDefault
	ColorRed     = tcell.StyleDefault.Foreground(tcell.ColorRed)
	ColorGreen   = tcell.StyleDefault.Foreground(tcell.ColorGreen)
	ColorBlue    = tcell.StyleDefault.Foreground(tcell.ColorBlue)
	ColorCyan    = tcell.StyleDefault.Foreground(tcell.ColorAqua)
)

func NewScreen() (*Screen, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err = s.Init(); err != nil {
		return nil, err
	}
	// s.Clear()
	return &Screen{screen: s}, nil
}

func (s *Screen) Close() {
	if s.screen != nil {
		s.screen.Fini()
	}
}

func (s *Screen) Clear() {
	if s.screen != nil {
		s.screen.Clear()
	}
}

func (s *Screen) Show() {
	if s.screen != nil {
		s.screen.Show()
	}
}

func (s *Screen) Size() (int, int) {
	if s.screen != nil {
		return s.screen.Size()
	}
	return 80, 24
}

func (s *Screen) drawText(x, y int, text string, style tcell.Style) {
	for i, r := range text {
		s.screen.SetContent(x+i, y, r, nil, style)
	}
}

func (s *Screen) ShowAtBottom(message string, style tcell.Style) {
	s.Clear()
	_, height := s.Size()
	s.drawText(0, height-1, message, style)
	s.Show()
}

func (s *Screen) ShowWithContext(context, message string, msgStyle, ctxStyle tcell.Style) {
	s.Clear()
	if context != "" {
		s.drawText(0, 0, context, ctxStyle)
	}
	_, height := s.Size()
	s.drawText(0, height-1, message, msgStyle)
	s.Show()
}

type Logger struct {
	screen *Screen
}

func NewLogger() (*Logger, error) {
	screen, err := NewScreen()
	if err != nil {
		return &Logger{}, nil
	}
	return &Logger{screen: screen}, nil
}

func (l *Logger) Close() {
	if l.screen != nil {
		l.screen.Close()
	}
}

func (l *Logger) Clear() {
	if l.screen != nil {
		l.screen.Clear()
		l.screen.Show()
	}
}

func (l *Logger) output(message string, style tcell.Style, isError bool) {
	if l.screen != nil {
		l.screen.ShowAtBottom(message, style)
	}
	if isError {
		fmt.Fprintf(os.Stderr, "%s\n", message)
	} else {
		fmt.Println(message)
	}
}

func (l *Logger) ShowPrompt(message string) {
	if l.screen != nil {
		l.screen.ShowAtBottom(message, ColorBlue)
	}
	fmt.Print(message)
}

func (l *Logger) ShowInfo(message string)    { l.output(message, ColorDefault, false) }
func (l *Logger) ShowError(message string)   { l.output("Error: "+message, ColorRed, true) }
func (l *Logger) ShowSuccess(message string) { l.output(message, ColorGreen, false) }
func (l *Logger) ShowMessage(message string) { l.output(message, ColorDefault, false) }
func (l *Logger) ShowStatus(message string)  { l.output(message, ColorCyan, false) }
func (l *Logger) Error(message string)       { l.output("Error: "+message, ColorRed, true) }

func (l *Logger) ShowPromptWithContext(context, prompt string) {
	if l.screen != nil {
		l.screen.ShowWithContext(context, prompt, ColorBlue, ColorDefault)
	}
	if context != "" {
		fmt.Println(context)
	}
	fmt.Print(prompt)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if l.screen != nil {
		l.screen.ShowAtBottom(message, ColorDefault)
	}
	fmt.Print(message)
}

func (l *Logger) Println(args ...interface{}) {
	message := fmt.Sprintln(args...)
	if l.screen != nil {
		l.screen.ShowAtBottom(message, ColorDefault)
	}
	fmt.Print(message)
}
