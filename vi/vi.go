package vi // import "fortio.org/gvi/vi"

import (
	"bytes"
	"strings"

	"fortio.org/terminal/ansipixels"
)

type Mode int

const (
	NavMode Mode = iota
	CommandMode
	InsertMode
)

func (m Mode) String() string {
	switch m {
	case NavMode:
		return ansipixels.Cyan + "Navigation" + ansipixels.White
	case CommandMode:
		return ansipixels.Yellow + "Command" + ansipixels.White
	case InsertMode:
		return ansipixels.Green + "Insert" + ansipixels.White
	default:
		return "Unknown"
	}
}

type Vi struct {
	cmdMode      Mode
	ap           *ansipixels.AnsiPixels
	filename     string // Not used in this example, but could be used to track the file being edited
	cx, cy       int    // Cursor position
	inputBuf     []byte // Buffer for partial input
	buf          Buffer
	splash       bool // Show splash screen on first refresh.
	offset       int  // Offset in lines for scrolling.
	usableHeight int  // v.ap.H - 2
	keepMessage  bool // Clear command/message line after processing input or not.
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
	v.Update()
	return nil
}

func (v *Vi) Update() {
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
	v.ap.WriteAt(0, v.usableHeight, "%s %sFile: %s (%d/%d lines) - %s - @%d,%d [%dx%d] %s",
		ansipixels.Inverse, dirty, v.filename, v.cy+1+v.offset, v.buf.NumLines(), v.cmdMode.String(), v.cx+1, v.cy+1, v.ap.W, v.ap.H,
		ansipixels.Reset)
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

func (v *Vi) VScroll(delta int) {
	v.cy += delta
	if v.cy < 0 {
		v.offset = max(0, v.offset+v.cy)
		v.cy = 0
		v.Update() // only if we scrolled. (in theory... shouldn't update at <0 etc).
	} else if v.cy >= v.usableHeight {
		v.offset = min(v.buf.NumLines()-v.usableHeight, v.offset+v.cy-v.usableHeight+1)
		v.cy = v.usableHeight - 1 // Keep cursor within bounds
		v.Update()
	}
}

func (v *Vi) Beep() {
	v.ap.WriteRune('\a') // Beep for unrecognized command or error
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
		v.offset += v.cy - v.usableHeight/2 // Center the view
		v.cy = v.usableHeight / 2           // Center cursor vertically
		v.Update()
	case 'h', 0x7f: // Backspace or 'h'
		v.cx = max(0, v.cx-1) // Move cursor left
	case 'l':
		v.cx = min(v.ap.W-1, v.cx+1) // Move cursor right
	case 'i':
		v.cmdMode = InsertMode
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

func (v *Vi) command(data []byte) bool {
	cmd := string(data)
	cont := true
	switch cmd {
	case "q!":
		v.ap.WriteAt(0, v.ap.H-1, "Exiting without saving...\r\n")
		cont = false // Exit the editor
	case "q":
		if v.buf.IsDirty() {
			v.ap.WriteAt(0, v.ap.H-1, "Use :wq to save and exit. :q! to exit without saving.")
		} else {
			cont = false
			v.ap.WriteAt(0, v.ap.H-1, "Exiting...\r\n")
		}
	case "wq":
		cont = false
		fallthrough
	case "w":
		if !v.buf.IsDirty() {
			v.ap.WriteAt(0, v.ap.H-1, "No changes to save.")
		} else {
			err := v.buf.Save() // Save the buffer to the file
			if err != nil {
				v.ShowError("Error saving file", err)
				cont = true // Stay in command mode
			} else {
				v.ap.WriteAt(0, v.ap.H-1, "File saved successfully.")
				v.cmdMode = NavMode  // Switch back to navigation mode
				v.keepMessage = true // Keep the message on the status line
				v.UpdateStatus()     // Update status after saving
			}
		}
	default:
		v.ap.WriteAt(0, v.ap.H-1, "Unknown command: %q (:q to quit)", cmd)
	}
	return cont // Exit or Continue processing
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
		if r < 32 || r == 127 { // Filter out control characters and backspace
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
	cont := true
	if len(v.ap.Data) == 0 {
		return true // No input, continue
	}
	v.splash = false                              // No splash screen after first input
	v.inputBuf = append(v.inputBuf, v.ap.Data...) // Append new data to buffer
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
	case InsertMode:
		// Handle insert mode input (e.g., add to buffer)
		str := string(v.inputBuf) // probably ansiclean actually
		hasEsc := v.HasEsc()
		if hasEsc >= 0 {
			v.cmdMode = NavMode                // Switch back to navigation mode on escape
			str = str[:hasEsc]                 // Get the string up to the escape character
			v.inputBuf = v.inputBuf[hasEsc+1:] // Remove the escape sequence
			if len(v.inputBuf) == 0 {
				break // No input left before escape
			}
		} else {
			v.inputBuf = nil
		}
		// split by line (\r)
		retPos := strings.IndexByte(str, '\r')
		if retPos >= 0 {
			str = str[:retPos] // Remove everything after the first carriage return
			// TODO: handle str[retPos+1:]
			if len(str) == 0 {
				v.cy++
				v.cx = 0
				v.UpdateStatus()
				break // No input to insert, just move cursor down without beeping for empty input
			}
		}
		err := v.Insert(FilterSpecialChars(str)) // Insert the string into the buffer
		if err != nil {
			v.ShowError("Error inserting text", err)
			return false // abort
		}
		if retPos >= 0 {
			v.cy++
			v.cx = 0
		}
		v.UpdateStatus()
	}
	return cont // Continue processing or not if command was 'q'
}

func (v *Vi) Insert(str string) (err error) {
	if len(str) == 0 {
		v.Beep()   // only special characters/controls.
		return nil // Nothing to insert
	}
	v.buf.InsertChars(v.cy+v.offset, v.cx, str) // Insert the string at the current cursor position
	v.ap.WriteAtStr(v.cx, v.cy, str)
	v.cx, v.cy, err = v.ap.ReadCursorPosXY()
	return err
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
