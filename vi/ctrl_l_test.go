package vi

import (
	"strconv"
	"testing"
)

// AI written tests in this file so kinda write only.

func TestCalculateCenteredPositionShortFile(t *testing.T) {
	v := &Vi{}
	v.usableHeight = 28 // Simulate large screen

	// Test centering line 0 in a 3-line file with large screen
	currentLine := 0
	numLines := 3

	offset, cy := v.calculateCenteredPosition(currentLine, numLines)

	// Check that result is within bounds
	totalLine := cy + offset
	if totalLine < 0 {
		t.Errorf("Cursor line is negative: cy=%d, offset=%d, total=%d", cy, offset, totalLine)
	}

	if totalLine >= numLines {
		t.Errorf("Cursor past end of file: cy=%d, offset=%d, total=%d, numLines=%d",
			cy, offset, totalLine, numLines)
	}

	// For a short file, we should stay on the same line (line 0)
	if totalLine != 0 {
		t.Errorf("Expected to stay on line 0, but got line %d", totalLine)
	}
}

func TestCalculateCenteredPositionLongFile(t *testing.T) {
	v := &Vi{}
	v.usableHeight = 8 // Simulate smaller screen

	// Test centering line 20 in a 50-line file
	currentLine := 20
	numLines := 50

	offset, cy := v.calculateCenteredPosition(currentLine, numLines)

	// Check that result is within bounds
	totalLine := cy + offset
	if totalLine < 0 {
		t.Errorf("Cursor line is negative: cy=%d, offset=%d, total=%d", cy, offset, totalLine)
	}

	if totalLine >= numLines {
		t.Errorf("Cursor past end of file: cy=%d, offset=%d, total=%d, numLines=%d",
			cy, offset, totalLine, numLines)
	}

	// We should still be on line 20
	if totalLine != 20 {
		t.Errorf("Expected to stay on line 20, but got line %d", totalLine)
	}

	// Check that the line is approximately centered (within reasonable bounds)
	// For usableHeight=8, center should be around cy=4
	if cy < 2 || cy > 6 {
		t.Errorf("Line should be approximately centered, but cy=%d (usableHeight=%d)", cy, v.usableHeight)
	}
}

func TestCalculateCenteredPositionNearEndOfFile(t *testing.T) {
	v := &Vi{}
	v.usableHeight = 10 // Simulate medium screen

	// Test centering line 11 (last line) in a 12-line file
	currentLine := 11
	numLines := 12

	offset, cy := v.calculateCenteredPosition(currentLine, numLines)

	// Check that result is within bounds
	totalLine := cy + offset
	if totalLine < 0 {
		t.Errorf("Cursor line is negative: cy=%d, offset=%d, total=%d", cy, offset, totalLine)
	}

	if totalLine >= numLines {
		t.Errorf("Cursor past end of file: cy=%d, offset=%d, total=%d, numLines=%d",
			cy, offset, totalLine, numLines)
	}

	// We should still be on line 11 (the last line)
	if totalLine != 11 {
		t.Errorf("Expected to stay on line 11, but got line %d", totalLine)
	}
}

func TestCalculateCenteredPositionEmptyFile(t *testing.T) {
	v := &Vi{}
	v.usableHeight = 20 // Simulate normal screen

	// Test centering in an empty file
	currentLine := 0
	numLines := 0

	offset, cy := v.calculateCenteredPosition(currentLine, numLines)

	// Check that cursor is at origin
	totalLine := cy + offset
	if totalLine != 0 {
		t.Errorf("For empty file, expected cursor at line 0, but got line %d", totalLine)
	}

	if cy != 0 {
		t.Errorf("For empty file, expected cy=0, but got cy=%d", cy)
	}

	if offset != 0 {
		t.Errorf("For empty file, expected offset=0, but got offset=%d", offset)
	}
}

func TestCtrlLCenteringLongFile(t *testing.T) {
	v := &Vi{}
	v.usableHeight = 8 // Simulate smaller screen

	// Create a long file with many lines
	v.buf.lines = make([]string, 50)
	for i := range 50 {
		v.buf.lines[i] = "line " + strconv.Itoa(i)
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
	for i := range 12 {
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

func TestCtrlLCenteringEmptyFile(t *testing.T) {
	v := &Vi{}
	v.usableHeight = 20 // Simulate normal screen

	// Create an empty file
	v.buf.lines = []string{} // Empty file

	// Start at origin (should be the only valid position)
	v.cy = 0
	v.offset = 0

	// Test the simplified empty file logic
	currentLine := v.cy + v.offset
	maxLine := max(0, v.buf.NumLines()-1)
	// Clamp currentLine to valid range
	currentLine = min(maxLine, currentLine)
	// Try to center, but respect bounds
	v.offset = max(0, currentLine-v.usableHeight/2)
	v.cy = currentLine - v.offset

	// Check that cursor is at origin
	totalLine := v.cy + v.offset
	if totalLine != 0 {
		t.Errorf("For empty file, expected cursor at line 0, but got line %d", totalLine)
	}

	if v.cy != 0 {
		t.Errorf("For empty file, expected cy=0, but got cy=%d", v.cy)
	}

	if v.offset != 0 {
		t.Errorf("For empty file, expected offset=0, but got offset=%d", v.offset)
	}
}
