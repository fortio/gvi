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
	v.ap.WriteAt(0, v.usableHeight, "%s File: %s (%d/%d lines) - %s - @%d,%d [%dx%d] %s",
		ansipixels.Inverse, v.filename, v.cx+1+v.offset, v.buf.NumLines(), v.cmdMode.String(), v.cx+1, v.cy+1, v.ap.W, v.ap.H,
		ansipixels.Reset)
	v.ap.ClearEndOfLine()
	if v.cmdMode == CommandMode {
		v.CommandStatus()
	} else {
		v.ap.MoveCursor(0, v.ap.H-1)
		v.ap.ClearEndOfLine()
		v.ap.MoveCursor(v.cx, v.cy)
	}
}

func (v *Vi) VScroll(delta int) {
	v.cy += delta
	if v.cy < 0 {
		v.cy = 0
		v.offset = max(0, v.offset-1)
		v.Update() // only if we scrolled. (in theory... shouldn't update at <0 etc).
	} else if v.cy >= v.usableHeight {
		v.cy = v.usableHeight - 1 // Keep cursor within bounds
		v.offset = min(v.buf.NumLines()-v.usableHeight, v.offset+1)
		v.Update()
	}
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
	case 'h':
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
		v.ap.WriteRune('\a') // Beep for unrecognized command
	}
}

func (v *Vi) command(data []byte) bool {
	cmd := string(data)
	switch cmd {
	case "q":
		v.ap.WriteAt(0, v.ap.H-1, "Exiting...\r\n")
		return false // Exit the editor
	case "w":
		v.ap.WriteAt(0, v.ap.H-1, "Write command not implemented yet")
	default:
		v.ap.WriteAt(0, v.ap.H-1, "Unknown command: %q (:q to quit)", cmd)
	}
	// for now stay in command mode until esc
	return true // Continue processing
}

func (v *Vi) HasEsc() int {
	// Check if the buffer contains an escape character
	return bytes.IndexByte(v.inputBuf, '\x1b')
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
		} else {
			v.inputBuf = nil
		}
		retPos := strings.IndexByte(str, '\r')
		if retPos >= 0 {
			str = str[:retPos] // Remove everything after the first carriage return
		}
		// split by line (\r)
		v.ap.WriteAtStr(v.cx, v.cy, str)
		if retPos >= 0 {
			v.cy++
			v.cx = 0
		} else {
			v.cx += len(str) // Move cursor right by the length of the input
		}
		v.UpdateStatus()
	}
	return cont // Continue processing or not if command was 'q'
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
