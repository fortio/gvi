package vi // import "fortio.org/gvi/vi"

import (
	"bytes"
	"fmt"
	"strings"

	"fortio.org/terminal/ansipixels"
)

type Mode int

const (
	NavMode Mode = iota
	CommandMode
	InsertMode
	AppendMode
)

func (m Mode) String() string {
	switch m {
	case NavMode:
		return ansipixels.Cyan + "Navigation" + ansipixels.White
	case CommandMode:
		return ansipixels.Yellow + "Command" + ansipixels.White
	case InsertMode:
		return ansipixels.Green + "Insert" + ansipixels.White
	case AppendMode:
		return ansipixels.Green + "Append" + ansipixels.White
	default:
		return "Unknown"
	}
}

type Vi struct {
	cmdMode        Mode
	ap             *ansipixels.AnsiPixels
	filename       string // Not used in this example, but could be used to track the file being edited
	cx, cy         int    // Cursor position
	inputBuf       []byte // Buffer for partial input
	buf            Buffer
	splash         bool // Show splash screen on first refresh.
	offset         int  // Offset in lines for scrolling.
	usableHeight   int  // v.ap.H - 2
	keepMessage    bool // Clear command/message line after processing input or not.
	tabs           []int
	Debug          bool // Debug mode flag
	fullRefresh    int  // Counter for full screen refreshes
	screenWidthCnt int  // Counter for ScreenWidth calls
	screenAtCnt    int  // Counter for ScreenAtToRune calls
}

func NewVi(ap *ansipixels.AnsiPixels) *Vi {
	return &Vi{
		cmdMode:      NavMode,
		ap:           ap,
		filename:     "...", // no filename case.
		splash:       true,  // Show splash screen on first refresh.
		usableHeight: ap.H - 2,
	}
}

func (v *Vi) UpdateRS() error {
	v.usableHeight = v.ap.H - 2
	v.UpdateTabs()
	v.Update()
	return nil
}

func (v *Vi) Update() {
	v.fullRefresh++ // Increment full refresh counter
	v.ap.StartSyncMode()
	v.ap.ClearScreen()
	lines := v.buf.GetLines(v.offset, v.usableHeight) // Get the lines from the buffer and display them
	for i, line := range lines {
		v.ap.WriteAtStr(0, i, line)
	}
	v.UpdateStatus()
	if v.splash {
		v.ap.WriteBoxed(v.ap.H/2-4, "Welcome to gvi (vi in go)!\n'ESC:q' to quit\nhjkl to move\nEsc, i, : to switch mode\ntry resize\n")
	}
}

func (v *Vi) CommandStatus() {
	v.ap.WriteAt(0, v.ap.H-1, ":%s", string(v.inputBuf))
	v.ap.ClearEndOfLine()
}

func (v *Vi) UpdateStatus() {
	dirty := ""
	if v.buf.IsDirty() {
		dirty = ansipixels.Purple + "*" + ansipixels.White
	}
	debugInfo := ""
	if v.Debug {
		debugInfo = fmt.Sprintf(" F:%d SW:%d SA:%d", v.fullRefresh, v.screenWidthCnt, v.screenAtCnt)
	}
	v.ap.WriteAt(0, v.usableHeight, "%s %sFile: %s (%d/%d lines) - %s - @%d,%d [%dx%d]%s %s",
		ansipixels.Inverse, dirty, v.filename, v.cy+1+v.offset, v.buf.NumLines(),
		v.cmdMode.String(), v.cx+1, v.cy+1, v.ap.W, v.ap.H, debugInfo, ansipixels.Reset)
	v.ap.ClearEndOfLine()
	if v.cmdMode == CommandMode {
		v.CommandStatus()
	} else {
		if !v.keepMessage {
			v.ap.MoveCursor(0, v.ap.H-1)
			v.ap.ClearEndOfLine()
		}
		v.ap.MoveCursor(v.cx, v.cy)
		v.keepMessage = false // Clear status line only if not in command mode
	}
}

// VScrollWithoutUpdate moves the cursor and handles scrolling but doesn't force a full screen update.
// Use this when you only need cursor movement and will handle the display update separately.
func (v *Vi) VScrollWithoutUpdate(delta int) bool {
	v.cy += delta
	scrolled := false
	if v.cy < 0 {
		v.offset = max(0, v.offset+v.cy)
		v.cy = 0
		scrolled = true
	} else if v.cy >= v.usableHeight {
		v.offset = min(v.buf.NumLines()-v.usableHeight, v.offset+v.cy-v.usableHeight+1)
		v.cy = v.usableHeight - 1 // Keep cursor within bounds
		scrolled = true
	}
	return scrolled // Return true if scrolling occurred
}

func (v *Vi) VScroll(delta int) {
	if v.VScrollWithoutUpdate(delta) {
		v.Update() // only if we scrolled. (in theory... shouldn't update at <0 etc).
	}
}

func (v *Vi) BufferLineNumber() int {
	return v.cy + v.offset
}

func (v *Vi) Beep() {
	v.ap.WriteRune('\a') // Beep for unrecognized command or error
}

// calculateCenteredPosition returns the offset and cy values needed to center currentLine
// on the screen. This is a pure function with no side effects.
func (v *Vi) calculateCenteredPosition(currentLine, numLines int) (offset, cy int) {
	maxLine := max(0, numLines-1) // file might be empty, let's not have -1 as last line.
	// Clamp currentLine to valid range
	currentLine = min(maxLine, currentLine)
	// Try to center, but respect bounds
	offset = max(0, currentLine-v.usableHeight/2)
	cy = currentLine - offset
	return offset, cy
}

func (v *Vi) AppendModeOn() {
	v.cmdMode = AppendMode
}

func (v *Vi) InsertModeOn() {
	v.cmdMode = InsertMode
}

// Append() returns true if we are in optimized append mode (vs regular insert mode).
func (v *Vi) Append() bool {
	return v.cmdMode == AppendMode
}

func (v *Vi) navigate(b byte) {
	// scroll instead when reading edges
	switch b {
	case 'j':
		v.VScroll(1) // Move cursor down
	case 'k':
		v.VScroll(-1) // Move cursor up
	case 4: // Ctrl-D
		v.VScroll(v.usableHeight / 2) // Half page down
	case 21: // Ctrl-U
		v.VScroll(-v.usableHeight / 2) // Half page up
	case 6: // Ctrl-F
		v.VScroll(v.usableHeight) // Page down
	case 2: // Ctrl-B
		v.VScroll(-v.usableHeight) // Page up
	case 12: // Ctrl-L - do like emacs and also recenter so we don't need "zz" for now
		// Center current line, with bounds checking
		v.offset, v.cy = v.calculateCenteredPosition(v.BufferLineNumber(), v.buf.NumLines())
		v.Update()
	case 'h', 0x7f: // Backspace or 'h'
		v.cx = max(0, v.cx-1) // Move cursor left
	case 'l':
		v.cx = min(v.ap.W-1, v.cx+1) // Move cursor right
	case 'i':
		if v.cx == 0 && v.EmptyLine() {
			v.AppendModeOn() // really append (eg initial empty line and hit 'i')
		} else {
			v.InsertModeOn()
		}
	case 'o': // new line below
		v.AppendModeOn()
		v.handleNewlineInsertion()
	case 'O': // new line above
		v.AppendModeOn()
		v.cy-- // need to work on first line too - no clamping.
		v.handleNewlineInsertion()
	case '$':
		// Move to end of line
		v.cx = max(0, v.ScreenWidth(v.buf.GetLine(v.BufferLineNumber()))-1) // Move cursor to end of line
	case '0':
		// Move to start of line
		v.cx = 0 // Move cursor to start of line
	case 'G':
		// Go to the last line
		v.cx = 0 // Reset cursor to start of line
		v.VScroll(v.buf.NumLines() - v.BufferLineNumber())
	case 'A':
		// Append at end of line
		currentLine := v.buf.GetLine(v.BufferLineNumber())
		v.cx = v.ScreenWidth(currentLine) // Move cursor to end of line
		v.AppendModeOn()                  // We're now in append mode
	case ':':
		v.cmdMode = CommandMode
		v.ap.WriteAtStr(0, v.ap.H-1, ":")
		v.ap.ClearEndOfLine() // Clear the command line
	case 0x1b: // Escape key
		// nothing to do, it's ok
	default:
		// beep
		v.Beep() // Beep for unrecognized command
	}
}

// EmptyLine checks if the current line is empty.
func (v *Vi) EmptyLine() bool {
	return v.buf.GetLine(v.cy+v.offset) == ""
}

func (v *Vi) WriteBottom(msg string, args ...any) {
	v.ap.WriteAt(0, v.ap.H-1, msg, args...)
}

func (v *Vi) CmdResult(msg string, args ...any) {
	v.WriteBottom(msg, args...)
	v.cmdMode = NavMode  // Switch back to navigation mode
	v.keepMessage = true // Keep the message on the status line
	v.UpdateStatus()     // Update status after saving
}

func (v *Vi) command(data []byte) bool {
	cmd := string(data)
	cont := true
	overwrite := true
	msg := "Error overwriting file"
	switch {
	case cmd == "q!":
		v.ap.WriteAt(0, v.ap.H-1, "Exiting without saving...\r\n")
		cont = false // Exit the editor
	case cmd == "q":
		if v.buf.IsDirty() {
			v.WriteBottom("Use :wq to save and exit. :q! to exit without saving.")
		} else {
			cont = false
			v.WriteBottom("Exiting...\r\n")
		}
	case cmd == "wq":
		cont = false
		fallthrough
	case cmd == "w":
		if !v.buf.IsDirty() {
			v.WriteBottom("No changes to save.")
		} else {
			cont = v.Save()
		}
	case cmd == "tabs":
		// v.UpdateTabs() // done on resize already.
		v.CmdResult("Tabs: %v", v.tabs)
	case strings.HasPrefix(cmd, "w "):
		overwrite = false
		msg = "Error opening new file (use :w! to overwrite): "
		fallthrough
	case strings.HasPrefix(cmd, "w! "):
		fname := cmd[strings.IndexByte(cmd, ' ')+1:] // Get the filename after "w " or "w! "
		err := v.buf.OpenNewFile(fname, overwrite)
		if err != nil {
			v.ShowError(msg, err)
			break
		}
		v.filename = fname // Update the filename in the editor
		_ = v.Save()
	default:
		v.WriteBottom("Unknown command: %q (:q to quit)", cmd)
	}
	return cont // Exit or Continue processing
}

func (v *Vi) Save() (cont bool) {
	err := v.buf.Save() // Save the buffer to the file
	if err != nil {
		v.ShowError("Error saving file", err)
		cont = true // Stay in command mode
	} else {
		// TODO: in common with tabs etc... make a function to display result yet switch back to nav mode
		v.CmdResult("File saved successfully.")
	}
	return
}

func (v *Vi) HasEsc() int {
	// Check if the buffer contains an escape character
	return bytes.IndexByte(v.inputBuf, '\x1b')
}

func (v *Vi) hasBackspace() int {
	// Check if the buffer contains a backspace character
	return bytes.IndexByte(v.inputBuf, '\x7f')
}

func FilterSpecialChars(str string) string {
	// iterate over the string and filter out special characters
	changed := false
	var runes []rune
	orunes := []rune(str) // Convert string to rune slice for proper handling of Unicode characters
	for i, r := range orunes {
		if (r < 32 && r != '\t') || r == 127 { // Filter out control characters and backspace but not tab.
			if !changed {
				changed = true                         // We are changing the string
				runes = make([]rune, 0, len(orunes)-1) // Initialize runes with capacity
				runes = append(runes, orunes[:i]...)   // Remove the special character
				continue                               // Skip this character
			}
			continue
		}
		if changed {
			runes = append(runes, r)
		}
	}
	if !changed {
		return str // No changes, return original string
	}
	return string(runes)
}

func (v *Vi) Process() bool {
	// Process the input buffer and update the state
	cont := true
	if len(v.ap.Data) == 0 {
		return cont // No input, continue
	}
	if v.splash {
		v.splash = false // No splash screen after first input
		v.Update()
	}
	v.inputBuf = append(v.inputBuf, v.ap.Data...) // Append new data to buffer
	for len(v.inputBuf) > 0 {
		cont = v.ProcessOne()
		// command mode does leave currently typed so far input in the inputBuf but other modes
		// need to consume all input (like a large paste in insert mode)
		if !cont || v.cmdMode == CommandMode {
			break
		}
	}
	return cont // Continue processing or not if command was 'q'
}

func (v *Vi) ProcessOne() bool {
	cont := true
	switch v.cmdMode {
	case NavMode:
		c := v.inputBuf[0]
		v.inputBuf = v.inputBuf[1:] // Remove the first byte for processing
		v.navigate(c)
		v.UpdateStatus()
	case CommandMode:
		v.inputBuf = bytes.TrimPrefix(v.inputBuf, []byte{':'}) // Remove extra leading ':', useful after error.
		hasBackspace := v.hasBackspace()
		if hasBackspace == 0 {
			v.cmdMode = NavMode // Switch back to navigation mode if backspace is pressed in command mode
			v.ap.MoveCursor(v.cx, v.cy)
			v.inputBuf = nil
			v.UpdateStatus()
			return true // Continue processing
		}
		if hasBackspace > 0 {
			// erase character before the backspace and the backspace itself
			v.inputBuf = append(v.inputBuf[:hasBackspace-1], v.inputBuf[hasBackspace+1:]...)
		}
		v.UpdateStatus()
		hasEsc := v.HasEsc()
		if hasEsc >= 0 {
			v.cmdMode = NavMode                // Switch back to navigation mode on escape
			v.inputBuf = v.inputBuf[hasEsc+1:] // Remove the escape sequence
			v.ap.MoveCursor(0, v.ap.H-1)       // Move cursor to the command line
			v.ap.ClearEndOfLine()
			break
		}
		ret := bytes.IndexByte(v.inputBuf, '\r')
		if ret >= 0 {
			data := v.inputBuf[:ret]        // Get the command input up to the carriage return
			v.inputBuf = v.inputBuf[ret+1:] // Remove the command input from
			cont = v.command(data)
		}
	case InsertMode, AppendMode:
		// Handle insert mode input (e.g., add to buffer)
		str := string(v.inputBuf)
		hasEsc := v.HasEsc()
		if hasEsc >= 0 {
			v.cmdMode = NavMode                // Switch back to navigation mode on escape
			str = str[:hasEsc]                 // Get the string up to the escape character
			v.inputBuf = v.inputBuf[hasEsc+1:] // Remove the escape sequence
			if len(v.inputBuf) == 0 {
				v.UpdateStatus()
				break // No input left before escape
			}
		} else {
			v.inputBuf = nil
		}
		// split by line (\r)
		retPos := strings.IndexByte(str, '\r')
		if retPos >= 0 {
			v.inputBuf = []byte(str[retPos+1:]) // Keep the rest of the input after the carriage return
			str = str[:retPos]                  // Remove everything after the first carriage return
		}

		// Insert any text content first
		if len(str) > 0 {
			v.Insert(FilterSpecialChars(str))
		}

		// Handle newline if present
		if retPos >= 0 {
			v.handleNewlineInsertion()
			// After newline, we're at the beginning of a new line at the end of file
			// So we can stay in append mode if we were already in it
		} else {
			v.UpdateStatus() // Just update status if no newline
		}
	}
	return cont // Continue processing or not if command was 'q'
}

func (v *Vi) Insert(str string) {
	if len(str) == 0 {
		v.Beep() // only special characters/controls.
		return   // Nothing to insert
	}
	lineNum := v.BufferLineNumber()
	var line string
	if v.Append() {
		v.buf.AppendToLine(lineNum, str)
	} else {
		line = v.buf.InsertChars(v, lineNum, v.cx, str) // Insert the string at the current cursor position
	}
	v.ap.WriteAtStr(v.cx, v.cy, str)
	v.cx, v.cy, _ = v.ap.ReadCursorPosXY()
	if line == "" {
		v.AppendModeOn() // If we inserted at the end of the line, switch to cheaper append mode
	} else {
		v.ap.MoveHorizontally(0) // Move cursor to the start of the line
		v.ap.ClearEndOfLine()
		v.ap.WriteString(line) // Write the full line.
	}
}

// InsertNewline handles inserting a newline at the current cursor position.
// It splits the current line and creates a new line with the remaining text.
func (v *Vi) InsertNewline() {
	currentLineNum := v.BufferLineNumber()
	currentLine := v.buf.GetLine(currentLineNum)

	// Convert screen position to rune offset for proper Unicode handling
	runeOffset := v.ScreenAtToRune(v.cx, currentLine)

	v.InsertNewlineAtOffset(runeOffset, currentLineNum, currentLine)
}

// InsertNewlineAtOffset handles inserting a newline at a pre-calculated byte offset.
// This avoids the expensive screen-position-to-byte-offset calculation when the offset is already known.
func (v *Vi) InsertNewlineAtOffset(runeOffset, currentLineNum int, currentLine string) {
	// Split the line at the specified offset
	var leftPart, rightPart string
	if runeOffset <= len(currentLine) {
		leftPart = currentLine[:runeOffset]
		rightPart = currentLine[runeOffset:]
	} else {
		// Cursor is beyond the line, pad with spaces
		leftPart = currentLine + strings.Repeat(" ", runeOffset-len(currentLine))
	}
	// Update the current line with the left part
	v.buf.ReplaceLine(currentLineNum, leftPart)
	// Insert a new line with the right part
	v.buf.InsertLine(currentLineNum+1, rightPart)
}

// handleNewlineInsertion handles the insertion of a newline with optimized screen updates.
func (v *Vi) handleNewlineInsertion() {
	currentLineNum := v.BufferLineNumber()
	// Special case of insert at the top of file ('O' command at c.y==-1 offset 0)
	if currentLineNum < 0 {
		v.buf.InsertLine(0, "") // Insert a new line at the top
		v.cy = 0                // Reset cursor to the first line
		v.cx = 0                // Reset cursor to the start of the line
		v.Update()              // Update the display after inserting the new line
		return
	}
	currentLine := v.buf.GetLine(currentLineNum)

	var runeOffset int
	canFastUpdate := true

	if v.Append() {
		// In append mode, we're at the end of the line - no need to calculate screen position
		runeOffset = len(currentLine)
	} else {
		// Only calculate screen position if we're not in append mode
		runeOffset = v.ScreenAtToRune(v.cx, currentLine)
		// Check if we can do a fast update (no full screen redraw needed)
		canFastUpdate = (runeOffset >= len(currentLine)) // At or past end of line
	}
	canFastUpdate = canFastUpdate && (currentLineNum >= v.buf.NumLines()-1) // At or past end of file

	v.InsertNewlineAtOffset(runeOffset, currentLineNum, currentLine)
	scrolled := v.VScrollWithoutUpdate(1)
	v.cx = 0

	if scrolled || !canFastUpdate {
		v.Update() // Full update needed for scrolling or line splitting
	} else {
		// Fast update: status update only (cursor is already positioned correctly by VScrollWithoutUpdate)
		v.UpdateStatus()
	}
}

func (v *Vi) ShowError(msg string, err error) {
	v.ap.WriteAt(0, v.ap.H-1, "%s%s: %v%s", ansipixels.Red, msg, err, ansipixels.Reset)
}

func (v *Vi) Open(filename string) {
	v.filename = filename
	v.splash = false // No splash screen when opening a file
	v.UpdateStatus()
	err := v.buf.Open(filename)
	if err != nil {
		v.ShowError("Error opening file", err)
		return
	}
	v.Update()
	v.ap.WriteAt(0, v.ap.H-1, "%sOpened file: %s%s", ansipixels.Green, filename, ansipixels.Reset)
}
