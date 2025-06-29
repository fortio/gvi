// Package vi implements a simple vi / vim like editor in Go.
// Reusable library part of the gvi editor (cli).
package vi

import (
	"bufio"
	"os"
)

// Buffer represents a full buffer (file) in the editor.
// A view of it is shown in the terminal.
type Buffer struct {
	f     *os.File // File handle for the buffer
	lines [][]byte
}

// Open initializes the buffer with the contents of the file.
func (b *Buffer) Open(filename string) error {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	b.f = f
	// Split the file into lines
	s := bufio.NewScanner(f)
	for s.Scan() {
		b.lines = append(b.lines, s.Bytes())
	}
	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

// GetLines returns the lines in the buffer from start to end.
func (b *Buffer) GetLines(start, end int) [][]byte {
	if start < 0 {
		start = 0
	}
	if end > len(b.lines) {
		end = len(b.lines)
	}
	if start >= end {
		return nil // No lines to return
	}
	return b.lines[start:end]
}
