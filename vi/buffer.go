// Package vi implements a simple vi / vim like editor in Go.
// Reusable library part of the gvi editor (cli).
package vi

import (
	"bufio"
	"os"
	"strings"
)

// Buffer represents a full buffer (file) in the editor.
// A view of it is shown in the terminal.
type Buffer struct {
	f     *os.File // File handle for the buffer
	lines []string
	dirty bool // True if the buffer has unsaved changes
}

// Open initializes the buffer with the contents of the file.
func (b *Buffer) Open(filename string) error {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0o666)
	if err != nil {
		return err
	}
	b.f = f
	// Split the file into lines
	s := bufio.NewScanner(f)
	for s.Scan() {
		b.lines = append(b.lines, s.Text())
	}
	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

// GetLines returns the lines in the buffer from start to end.
func (b *Buffer) GetLines(start, num int) []string {
	if start < 0 {
		start = 0
	}
	end := start + num
	if end > len(b.lines) {
		end = len(b.lines)
	}
	if start >= end {
		return nil // No lines to return
	}
	return b.lines[start:end]
}

func (b *Buffer) Close() error {
	if b.f != nil {
		return b.f.Close()
	}
	return nil
}

func (b *Buffer) NumLines() int {
	return len(b.lines)
}

func (b *Buffer) IsDirty() bool {
	return b.dirty
}

func (b *Buffer) InsertLine(lineNum, at int, text string) {
	if lineNum < 0 || lineNum > len(b.lines) {
		return // Invalid line number
	}
	b.lines = append(b.lines[:lineNum], append([]string{text}, b.lines[lineNum:]...)...)
	b.dirty = true
}

func (b *Buffer) InsertChars(lineNum, at int, text string) {
	if lineNum < 0 || lineNum > len(b.lines) {
		return // Invalid line number
	}
	line := b.lines[lineNum]
	if at > len(line) {
		line = line + strings.Repeat(" ", at-len(line))
	}
	b.lines[lineNum] = line[:at] + text + line[at:]
	b.dirty = true
}
