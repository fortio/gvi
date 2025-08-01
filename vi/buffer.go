// Package vi implements a simple vi / vim like editor in Go.
// Reusable library part of the gvi editor (cli).
package vi

import (
	"bufio"
	"errors"
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

func (b *Buffer) InsertLine(lineNum int, text string) {
	if lineNum < 0 || lineNum > len(b.lines) {
		return // Invalid line number
	}
	b.lines = append(b.lines[:lineNum], append([]string{text}, b.lines[lineNum:]...)...)
	b.dirty = true
}

// Returns the full line if insert is in the middle, empty if that was already at the end.
// It makes most common typing (at the end/append) faster.
func (b *Buffer) InsertChars(v *Vi, lineNum, at int, text string) string {
	if lineNum < 0 {
		panic("negative line number")
	}
	b.dirty = true
	if lineNum >= len(b.lines) {
		for i := len(b.lines); i < lineNum; i++ {
			b.lines = append(b.lines, "")
		}
		b.lines = append(b.lines, strings.Repeat(" ", at)+text)
		return ""
	}
	line := b.lines[lineNum]
	// TODO: skip this (expensive stuff) when we're insert at end of line mode / remember.
	atOffset := v.ScreenAtToRune(at, line) // Convert screen position to byte offset
	returnLine := false
	if atOffset > len(line) {
		// We're inserting beyond the end of the line content, need padding
		paddingNeeded := atOffset - len(line)
		line += strings.Repeat(" ", paddingNeeded)
		atOffset = len(line) // Insert at end of padded line
	} else if atOffset < len(line) {
		returnLine = true // We are inserting in the middle of the line
	}
	line = line[:atOffset] + text + line[atOffset:]
	b.lines[lineNum] = line
	if returnLine {
		return line
	}
	return "" // was insert at the end, no line to return
}

// ReplaceLine replaces the content of a line at the given line number.
func (b *Buffer) ReplaceLine(lineNum int, newContent string) {
	if lineNum < 0 || lineNum >= len(b.lines) {
		panic("line number out of range")
	}
	b.lines[lineNum] = newContent
	b.dirty = true
}

// GetLine returns the content of a single line.
func (b *Buffer) GetLine(lineNum int) string {
	if lineNum < 0 || lineNum >= len(b.lines) {
		panic("line number out of range")
	}
	return b.lines[lineNum]
}

func (b *Buffer) Save() error {
	if b.f == nil {
		return errors.New("no file to save")
	}
	if !b.dirty {
		return nil // No changes to save
	}
	_, err := b.f.Seek(0, 0) // Reset file pointer to the beginning
	if err != nil {
		return err
	}
	var written int64
	var n int
	for _, line := range b.lines {
		if n, err = b.f.WriteString(line + "\n"); err != nil {
			return err
		}
		written += int64(n)
	}
	if err := b.f.Truncate(written); err != nil {
		return err
	}
	b.dirty = false // Reset dirty flag after saving
	return nil
}
