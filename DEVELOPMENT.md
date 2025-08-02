# GVI Development Guide

## Overview
GVI is a vi editor implementation in Go that handles Unicode text properly, including wide characters, multi-rune grapheme clusters, tabs, and control characters.

## Key Architecture Components

### Core Files
- `vi/position.go` - Text positioning logic, screen coordinate to byte offset translation
- `vi/buffer.go` - Text buffer manipulation and character insertion
- `vi/buffer_test.go` - Test suite for text insertion functionality
- `vi/tabs.go` - Tab stop management
- `vi/vi.go` - Main vi editor logic

### Critical Functions
- `ScreenAtToRune(x, str)` - Converts screen position to byte offset in string
- `ScreenWidth(str)` - Calculates screen width of string (handles wide chars, tabs)
- `iterateGraphemes(str, callback)` - Unified iteration over grapheme clusters
- `InsertChars(v, line, pos, chars)` - Inserts characters with proper positioning
- `NextTab(x)` - Calculates next tab stop position

## Unicode Handling

### Key Dependencies
- `fortio.org/terminal/ansipixels` - Low level terminal handling
- `github.com/rivo/uniseg` - Grapheme cluster segmentation (handles multi-rune characters)
- `fortio.org/log` - Debug logging framework

### Character Types Handled
1. **Regular ASCII** - Single byte, width 1
2. **Wide Characters** - CJK characters, emojis (width 2)
3. **Multi-rune Graphemes** - Complex emojis like 👩‍🚀 (multiple Unicode code points, width 2)
4. **Tab Characters** - Variable width based on tab stops
5. **Control Characters** - Width 0 (handled automatically by uniseg)

### Critical Implementation Details
- Always use grapheme cluster boundaries for text positioning
- Screen width ≠ byte length ≠ rune count
- Tab width is calculated relative to current screen position
- Control characters have zero screen width
- Wide characters occupy 2 screen columns
- ScreenWidth and ScreenAtToRune are **expensive** and we try to minimize the number of time they are called (see counters -debug mode)
- likewise Update is expensive and to be avoided if UpdateStatus and write and/or clearing a single line can achieve the update.

## Development Workflow

### Testing
```bash
# Run tests with debug logging (preferred method)
LOGGER_LEVEL=debug go test -count 1 -v ./...

# The -count 1 flag disables test caching (equivalent to go clean -testcache)
```

### Debug Environment
- Set `LOGGER_LEVEL=Debug` environment variable to see detailed positioning logs
- Use `go clean -testcache` or `-count 1` to ensure fresh test runs
- VS Code debugging works with environment variable configuration

### Test Structure
- `TestInsertSingleRune` - Simple character insertion (rune-by-rune)
- `TestInsertMultiRuneGraphemes` - Complex graphemes (manual cursor control)
- Tests verify character order preservation (e.g., "A乒乓" should not become "A乓乒")

## Common Bug Patterns

### Position Calculation Bugs
- **Byte vs screen position confusion**: Always distinguish between string byte offsets and screen coordinates
- **Grapheme cluster splitting**: Never split multi-rune graphemes during insertion

### Insertion Logic Bugs
- **Padding calculation**: Use screen width, not byte length for spacing
- **Wide character positioning**: Insert after complete grapheme cluster, not in middle
- **Tab stop calculation**: Tabs expand to next tab stop, not fixed width

## Code Patterns

### Iterating Over Text
Always use `iterateGraphemes` helper for consistent handling:
```go
v.iterateGraphemes(str, func(offset, screenOffset, prevScreenOffset, consumed int) bool {
    // offset: byte position in string
    // screenOffset: cumulative screen width to this point
    // prevScreenOffset: screen width before this element
    // consumed: bytes consumed by this grapheme/tab
    // return true to stop iteration early
    return false
})
```

### Position Translation
```go
// Screen position to byte offset
byteOffset := v.ScreenAtToRune(screenX, lineText)

// Calculate screen width
screenWidth := v.ScreenWidth(lineText)
```

### Error-Prone Areas
1. **Tab handling** - Special case in iteration, custom width calculation
2. **End-of-line insertion** - May need padding beyond string length
3. **Wide character boundaries** - Must respect grapheme cluster integrity
4. **State preservation** - uniseg requires state tracking across calls

## Performance Considerations
- `iterateGraphemes` eliminates code duplication between position/width calculations
- Callback interface passes `consumed` bytes to avoid redundant uniseg calls
- Control characters are handled efficiently by uniseg (no special casing needed)

## Future Maintenance Notes
- uniseg library handles Unicode standard updates automatically
- Tab stop configuration is customizable via `v.tabs` slice
- Debug logging shows detailed position calculations for troubleshooting
- Test cases cover edge cases: empty strings, single chars, complex graphemes
- All Unicode handling is centralized in position.go for maintainability
