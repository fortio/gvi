package vi

import (
	"fortio.org/log"
	"github.com/rivo/uniseg"
)

// iterateGraphemes iterates through a string, calling the provided function for each
// grapheme cluster, tab, or control character. The callback function receives:
// - offset: byte offset in the string where this element starts
// - screenOffset: cumulative screen width up to and including this element
// - prevScreenOffset: screen width before this element was processed
// - consumed: number of bytes consumed by this element
//
// If the callback returns true, iteration stops early and the current screenOffset is returned.
// If iteration completes normally, returns the total screen width of the string.
func (v *Vi) iterateGraphemes(str string, fn func(offset, screenOffset, prevScreenOffset, consumed int) bool) int {
	screenOffset := 0
	offset := 0
	state := -1 // Initial state for grapheme cluster iteration

	for offset < len(str) {
		prevScreenOffset := screenOffset
		var consumed int
		// Handle tab characters specially (tabs need custom width calculation)
		if str[offset] == '\t' {
			screenOffset = v.NextTab(screenOffset)
			width := screenOffset - prevScreenOffset
			log.LogVf("iterateGraphemes: offset=%d, for tab, screenOffset=%d, width=%d", offset, screenOffset, width)
			consumed = 1 // Tab is always 1 byte
			state = -1   // Reset state after tab character
		} else {
			// Handle all characters (including control chars) with uniseg
			cluster, _, width, newState := uniseg.FirstGraphemeClusterInString(str[offset:], state)
			state = newState
			screenOffset += width
			log.LogVf("iterateGraphemes: offset=%d, cluster=%q, screenOffset=%d, width=%d", offset, cluster, screenOffset, width)
			consumed = len(cluster) // Length of the grapheme cluster in bytes
		}
		if fn(offset, screenOffset, prevScreenOffset, consumed) {
			return screenOffset
		}
		offset += consumed // Move to the next grapheme cluster
	}

	return screenOffset
}

// Translate a screen position to the byte offset of the rune in the current line.
func (v *Vi) ScreenAtToRune(x int, str string) int {
	if x < 0 {
		panic("negative x coordinate")
	}
	if x == 0 {
		return 0 // No offset for the first rune
	}
	if len(str) == 0 {
		return x // No content, return x as the offset
	}

	var result int
	finalScreenOffset := v.iterateGraphemes(str, func(offset, screenOffset, prevScreenOffset, consumed int) bool {
		log.LogVf("ScreenAtToRune: x=%d, offset=%d, screenOffset=%d", x, offset, screenOffset)

		if screenOffset > x {
			// If x falls within this grapheme cluster, insert after it
			if prevScreenOffset < x {
				result = offset + consumed // Use the consumed bytes from iterator
				log.LogVf("ScreenAtToRune: x=%d falls within element at offset %d, inserting after (line %q)", x, offset, str)
			} else {
				result = offset // Return the offset if the screen position is reached
				log.LogVf("ScreenAtToRune: x=%d reached at offset %d (line %q)", x, offset, str)
			}
			return true // Stop iteration
		}
		return false // Continue iteration
	})

	if finalScreenOffset <= x {
		// We've reached the end of the string
		// If x equals the screen width, insert at the end
		// If x is beyond the screen width, return offset that encodes padding needed
		if x == finalScreenOffset {
			return len(str) // Insert at the very end
		}
		result = len(str) + (x - finalScreenOffset) // Encode padding amount
		log.LogVf("ScreenAtToRune: x=%d reached end (screen offset %d) (line %q): %d", x, finalScreenOffset, str, result)
	}

	return result
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

// ScreenWidth calculates the screen width of a string, properly handling
// tabs, control characters, and multi-rune grapheme clusters.
func (v *Vi) ScreenWidth(str string) int {
	return v.iterateGraphemes(str, func(_, _, _, _ int) bool {
		return false // Never stop iteration, just calculate the full width
	})
}
