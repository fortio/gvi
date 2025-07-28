package vi

import (
	"testing"
)

func TestInsertChars(t *testing.T) {
	v := &Vi{}
	v.tabs = []int{4, 8, 12, 16, 20} // Set tab stops
	// v.ap = ansipixels.NewAnsiPixels(0)

	tests := []string{
		"a\tb",
		"abc",
		"a🎉",
		"A乒乓B",
		"😀🎉",
	}
	for _, str := range tests {
		v.buf.lines = nil // Reset the buffer lines
		v.cx = 0          // Reset cursor x position
		// Simulate inserting a string one by one
		var line string
		runes := []rune(str)
		for i, r := range runes {
			line = v.buf.InsertChars(v, 0, v.cx, string(r))
			v.cx++
			if r == '\t' {
				v.cx = 4
			}
			if line != "" {
				t.Errorf("Expected empty line after inserting %q, got %q", string(r), line)
			}
			actual := v.buf.lines[0]
			expected := string(runes[:i+1])
			if actual != expected {
				t.Errorf("Expected %q got %q", expected, actual)
			}
		}
	}
}
