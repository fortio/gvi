package vi

import (
	"testing"
)

func TestInsertWithTabs(t *testing.T) {
	v := &Vi{}
	v.tabs = []int{4, 8, 12, 16, 20} // Set tab stops
	// v.ap = ansipixels.NewAnsiPixels(0)

	str := "a\tb"
	// Simulate inserting a string with a tab character one by one
	var line string
	for _, r := range str {
		line = v.buf.InsertChars(v, 0, v.cx, string(r))
		v.cx++
		if r == '\t' {
			v.cx = 4
		}
		if line != "" {
			t.Errorf("Expected empty line after inserting %q, got %q", string(r), line)
		}
	}
	actual := v.buf.lines[0]
	if actual != str {
		t.Errorf("Expected %q got %q", str, actual)
	}
}
