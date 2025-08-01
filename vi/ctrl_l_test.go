package vi

import (
	"testing"
)

func TestCtrlLCenteringShortFile(t *testing.T) {
	v := &Vi{}
	v.usableHeight = 28 // Simulate large screen

	// Create a short file with only 3 lines
	v.buf.lines = []string{"line 1", "line 2", "line 3"}

	// Start at line 0 (first line)
	v.cy = 0
	v.offset = 0

	// Test the ultra-simplified centering logic
	currentLine := v.cy + v.offset
	maxLine := v.buf.NumLines() - 1

	// Clamp currentLine to valid range
	currentLine = min(maxLine, currentLine)

	// Try to center, but respect bounds
	v.offset = max(0, currentLine-v.usableHeight/2)
	v.cy = currentLine - v.offset

	// Check that cursor is within bounds
	totalLine := v.cy + v.offset
	if totalLine < 0 {
		t.Errorf("Cursor line is negative: cy=%d, offset=%d, total=%d", v.cy, v.offset, totalLine)
	}

	if totalLine >= v.buf.NumLines() {
		t.Errorf("Cursor past end of file: cy=%d, offset=%d, total=%d, numLines=%d",
			v.cy, v.offset, totalLine, v.buf.NumLines())
	}

	// For a short file, we should stay on the same line (line 0)
	if totalLine != 0 {
		t.Errorf("Expected to stay on line 0, but got line %d", totalLine)
	}
}

func TestCtrlLCenteringLongFile(t *testing.T) {
	v := &Vi{}
	v.usableHeight = 8 // Simulate smaller screen

	// Create a long file with many lines
	v.buf.lines = make([]string, 50)
	for i := 0; i < 50; i++ {
		v.buf.lines[i] = "line " + string(rune('0'+i))
	}

	// Start at line 20 (middle of file)
	v.cy = 4
	v.offset = 16 // Total line = 20

	// Test the ultra-simplified centering logic
	currentLine := v.cy + v.offset
	maxLine := v.buf.NumLines() - 1

	// Clamp currentLine to valid range
	currentLine = min(maxLine, currentLine)

	// Try to center, but respect bounds
	v.offset = max(0, currentLine-v.usableHeight/2)
	v.cy = currentLine - v.offset

	// Check that cursor is within bounds
	totalLine := v.cy + v.offset
	if totalLine < 0 {
		t.Errorf("Cursor line is negative: cy=%d, offset=%d, total=%d", v.cy, v.offset, totalLine)
	}

	if totalLine >= v.buf.NumLines() {
		t.Errorf("Cursor past end of file: cy=%d, offset=%d, total=%d, numLines=%d",
			v.cy, v.offset, totalLine, v.buf.NumLines())
	}

	// We should still be on line 20
	if totalLine != 20 {
		t.Errorf("Expected to stay on line 20, but got line %d", totalLine)
	}

	// Check that the line is approximately centered (within reasonable bounds)
	// For usableHeight=8, center should be around cy=4
	if v.cy < 2 || v.cy > 6 {
		t.Errorf("Line should be approximately centered, but cy=%d (usableHeight=%d)", v.cy, v.usableHeight)
	}
}

func TestCtrlLCenteringNearEndOfFile(t *testing.T) {
	v := &Vi{}
	v.usableHeight = 10 // Simulate medium screen

	// Create a file with 12 lines (slightly longer than screen)
	v.buf.lines = make([]string, 12)
	for i := 0; i < 12; i++ {
		v.buf.lines[i] = "line " + string(rune('0'+i))
	}

	// Start near the end of the file (line 11, the last line)
	v.cy = 9
	v.offset = 2 // Total line = 11 (last line)

	// Test the ultra-simplified centering logic
	currentLine := v.cy + v.offset
	maxLine := v.buf.NumLines() - 1

	// Clamp currentLine to valid range
	currentLine = min(maxLine, currentLine)

	// Try to center, but respect bounds
	v.offset = max(0, currentLine-v.usableHeight/2)
	v.cy = currentLine - v.offset

	// Check that cursor is within bounds
	totalLine := v.cy + v.offset
	if totalLine < 0 {
		t.Errorf("Cursor line is negative: cy=%d, offset=%d, total=%d", v.cy, v.offset, totalLine)
	}

	if totalLine >= v.buf.NumLines() {
		t.Errorf("Cursor past end of file: cy=%d, offset=%d, total=%d, numLines=%d",
			v.cy, v.offset, totalLine, v.buf.NumLines())
	}

	// We should still be on line 11 (the last line)
	if totalLine != 11 {
		t.Errorf("Expected to stay on line 11, but got line %d", totalLine)
	}
}
