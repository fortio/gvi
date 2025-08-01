package vi

import (
	"testing"
)

func TestInsertSingleRune(t *testing.T) {
	v := &Vi{}
	v.tabs = []int{4, 8, 12, 16, 20} // Set tab stops

	// Test simple cases where each rune advances cursor by 1 (+ special case of tabs)
	simpleTests := []string{
		"a\tb",
		"a\tbc",
		"abc",
		"a🎉",
		"A乒乓B",
		"A乒乓BéC",
		"😀🎉",
		"😀🎉x",
		"a\001b", // Control character (Ctrl-A) with width 0
		"a\001bc",
	}
	for _, str := range simpleTests {
		v.buf.lines = nil // Reset the buffer lines
		v.cx = 0          // Reset cursor x position
		// Simulate inserting a string one by one
		var line string
		runes := []rune(str)
		for i, r := range runes {
			line = v.buf.InsertChars(v, 0, v.cx, string(r))
			switch {
			// approximation of 1 width for ascii and latin, works for what we have in tests and avoids
			// circular assumptions of using uniseq to test code that uses uniseg.
			case r > ' ' && r < 0x1100:
				v.cx++
			case r == '\t':
				v.cx = 4
			case r < ' ':
				// v.cx unchanged.
			default:
				v.cx += 2 // asian characters and smileys are double width
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

func TestInsertMultiRuneGraphemes(t *testing.T) {
	v := &Vi{}

	// Test complex multi-rune graphemes with manual cursor control
	// Test: "a👍🏽b" - thumbs up with skin tone modifier
	v.buf.lines = nil
	v.cx = 0

	// Insert 'a' at position 0
	v.buf.InsertChars(v, 0, v.cx, "a")
	v.cx = 1 // 'a' advances by 1

	// Insert '👍🏽' (complete grapheme) at position 1
	v.buf.InsertChars(v, 0, v.cx, "👍🏽")
	v.cx = 3 // '👍🏽' advances by 2 (it's a wide character)

	// Insert 'b' at position 3
	v.buf.InsertChars(v, 0, v.cx, "b")

	expected := "a👍🏽b"
	actual := v.buf.lines[0]
	if actual != expected {
		t.Errorf("Multi-rune test 1: Expected %q got %q", expected, actual)
	}

	// Test: "x👩‍🚀y" - woman astronaut (complex multi-rune grapheme)
	v.buf.lines = nil
	v.cx = 0

	// Insert 'x' at position 0
	v.buf.InsertChars(v, 0, v.cx, "x")
	v.cx = 1 // 'x' advances by 1

	// Insert complete grapheme '👩‍🚀' at position 1
	v.buf.InsertChars(v, 0, v.cx, "👩‍🚀")
	v.cx = 3 // '👩‍🚀' advances by 2 (it's a wide character)

	// Insert 'y' at position 3
	v.buf.InsertChars(v, 0, v.cx, "y")
	// by not incrementing v.cx it means go back to before 'y', and insert 'A'
	v.buf.InsertChars(v, 0, v.cx, "A")
	// and one more (to see if the issue is just about "the end" or any insert off by one)
	v.buf.InsertChars(v, 0, v.cx, "B")

	expected = "x👩‍🚀BAy"
	actual = v.buf.lines[0]
	if actual != expected {
		t.Errorf("Multi-rune test 2: Expected %q got %q", expected, actual)
	}

	// Test: Insert past the end of line (with padding)
	// Current line: "x👩‍🚀BAy" has screen width 6
	// Insert 'Z' at screen position 10 (beyond the end)
	v.cx = 10
	v.buf.InsertChars(v, 0, v.cx, "Z")

	expected = "x👩‍🚀BAy    Z" // 4 spaces of padding between 'y' and 'Z'
	actual = v.buf.lines[0]
	if actual != expected {
		t.Errorf("Past-end insert test: Expected %q got %q", expected, actual)
	}
}
