// Package vi implements a simple vi / vim like editor in Go.
// Reusable library part of the gvi editor (cli).
package vi

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

// ScreenPositionCalculator provides screen position to byte offset translation.
type ScreenPositionCalculator interface {
	ScreenAtToRune(x int, str string) int
}

/*
type Line struct {
	bytes []byte // Raw bytes of the line
}
*/

// Buffer represents a full buffer (file) in the editor.
// A view of it is shown in the terminal.
type Buffer struct {
	f     *os.File // File handle for the buffer
	lines []string
	dirty bool // True if the buffer has unsaved changes
}

// OpenNewFile doesn't overwrite existing files, returns false if file already exists.
func (b *Buffer) OpenNewFile(filename string, overwrite bool) error {
	mode := os.O_CREATE | os.O_WRONLY
	if !overwrite {
		mode |= os.O_EXCL
	}
	f, err := os.OpenFile(filename, mode, 0o644)
	if err != nil {
		return err
	}
	b.f = f
	return nil
}

// Open initializes the buffer with the contents of the file.
func (b *Buffer) Open(filename string) error {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0o644)
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
	end := min(start+num, len(b.lines))
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

// InsertChars returns the full line if insert is in the middle, empty if that was already at the end.
// It makes most common typing (at the end/append) faster.
// This is the expensive one (calculating screen offsets) but if it returns "" it means
// the insertion was at the of the line and subsequent appends can be done faster using AppendToLine.
func (b *Buffer) InsertChars(calc ScreenPositionCalculator, lineNum, at int, text string) string {
	if lineNum < 0 {
		panic("negative line number")
	}
	b.dirty = true
	// Pad with empty lines if inserting past the end of the buffer
	for lineNum >= len(b.lines) {
		b.lines = append(b.lines, "")
	}
	line := b.lines[lineNum]

	atOffset := calc.ScreenAtToRune(at, line) // Convert screen position to byte offset
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

// AppendToLine appends text to the end of a line.
// This is optimized for the common case of appending and skips screen position calculations.
func (b *Buffer) AppendToLine(lineNum int, text string) {
	if lineNum < 0 {
		panic("negative line number")
	}
	b.dirty = true
	// Pad with empty lines if inserting past the end of the buffer
	for lineNum >= len(b.lines) {
		b.lines = append(b.lines, "")
	}
	b.lines[lineNum] += text
}

// DeleteChar deletes a character at the specified screen position.
func (b *Buffer) DeleteChar(calc ScreenPositionCalculator, lineNum, at int) {
	if lineNum < 0 || lineNum >= len(b.lines) {
		return
	}
	line := b.lines[lineNum]
	if len(line) == 0 {
		return
	}

	byteOffset := calc.ScreenAtToRune(at, line)
	if byteOffset >= len(line) {
		return
	}

	// Find next rune boundary
	runes := []rune(line)
	for i := range runes {
		if len(string(runes[:i])) == byteOffset {
			b.lines[lineNum] = string(runes[:i]) + string(runes[i+1:])
			b.dirty = true
			return
		}
	}
}

// ReplaceLine replaces the content of a line at the given line number.
// Extends buffer with empty lines if necessary.
func (b *Buffer) ReplaceLine(lineNum int, newContent string) {
	if lineNum < 0 {
		panic("negative line number")
	}
	// Extend buffer if necessary
	for lineNum >= len(b.lines) {
		b.lines = append(b.lines, "")
	}
	b.lines[lineNum] = newContent
	b.dirty = true
}

// GetLine returns the content of a single line.
// Panics if lineNum is negative.
// Returns an empty string if lineNum is greater than or equal to the number of lines in the buffer.
func (b *Buffer) GetLine(lineNum int) string {
	if lineNum < 0 {
		panic("line number out of range")
	}
	if lineNum >= len(b.lines) {
		return "" // Return empty string if line number is out of range
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
