package vi

import (
	"fortio.org/log"
	"github.com/rivo/uniseg"
)

// Translate a screen position to the byte offset of the rune in the current line.
func (v *Vi) ScreenAtToRune(x int, str string) int {
	if x < 0 {
		panic("negative x coordinate")
	}
	if x == 0 {
		return 0 // No offset for the first rune
	}
	screenOffset := 0
	state := -1 // Initial state for grapheme cluster iteration
	var width int
	for offset, r := range str {
		switch {
		case r == '\t':
			// Handle tab characters by calculating their width
			screenOffset += v.NextTab(screenOffset)
		case r < ' ': // Shortcut for control characters: have no width.
			// nothing to do, just continue
		default: // other, assuming printable runes (maybe do ansiclean equivalent later)
			_, _, width, state = uniseg.FirstGraphemeClusterInString(str[offset:], state)
			screenOffset += width
		}
		if screenOffset > x {
			log.LogVf("ScreenAtToRune: x=%d reached at offset %d for rune %q (line %q)", x, offset, r, str)
			return offset // Return the offset if the screen position is reached
		}
	}
	res := len(str) + (x - screenOffset)
	log.LogVf("ScreenAtToRune: x=%d reached end (screen offset %d) (line %q): %d", x, screenOffset, str, res)
	return res
}

func (v *Vi) NextTab(x int) int {
	if len(v.tabs) == 0 {
		return x + 8
	}
	for _, tab := range v.tabs {
		if tab > x {
			return tab
		}
	}
	return x + 8 // If no tab is found, return the next tab position (default 8 spaces)
}
