package vi // import "fortio.org/gvi/vi"

import (
	"bytes"

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
		return "Navigation"
	case CommandMode:
		return "Command"
	case InsertMode:
		return "Insert"
	default:
		return "Unknown"
	}
}

type Vi struct {
	cmdMode  Mode
	ap       *ansipixels.AnsiPixels
	filename string // Not used in this example, but could be used to track the file being edited
	cx, cy   int    // Cursor position
	buf      []byte // Buffer for partial input
}

func NewVi(ap *ansipixels.AnsiPixels) *Vi {
	return &Vi{
		cmdMode:  NavMode,
		ap:       ap,
		filename: "...", // Placeholder for filename
	}
}

func (v *Vi) Update() error {
	v.ap.ClearScreen()
	v.UpdateStatus()
	v.ap.WriteBoxed(v.ap.H/2, "Hello, World!\nHiya caches\n':q' to quit\nEsc or I switch mode\ntry resize\n")
	return nil
}

func (v *Vi) UpdateStatus() {
	v.ap.WriteAt(0, v.ap.H-2, "%s File: %s - Mode: %s - %dx%d %s",
		ansipixels.Inverse, v.filename, v.cmdMode.String(), v.ap.W, v.ap.H,
		ansipixels.Reset)
	if v.cmdMode != CommandMode {
		v.ap.MoveCursor(v.cx, v.cy)
	}
}

func (v *Vi) navigate(b byte) {
	switch b {
	case 'j':
		v.cy++ // Move cursor down
	case 'k':
		v.cy-- // Move cursor up
	case 'h':
		v.cx-- // Move cursor left
	case 'l':
		v.cx++ // Move cursor right
	case 'i':
		v.cmdMode = InsertMode
	case ':':
		v.cmdMode = CommandMode
		v.ap.WriteAtStr(0, v.ap.H-1, ":")
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
	default:
		v.ap.WriteAt(0, v.ap.H-1, "Unknown command: %q (:q to quit)", cmd)
	}
	v.cmdMode = NavMode // Return to navigation mode after command
	return true         // Continue processing
}

func (v *Vi) HasEsc() int {
	// Check if the buffer contains an escape character
	return bytes.IndexByte(v.buf, '\x1b')
}

func (v *Vi) Process() bool {
	if len(v.ap.Data) == 0 {
		return true // No input, continue
	}
	v.buf = append(v.buf, v.ap.Data...) // Append new data to buffer
	switch v.cmdMode {
	case NavMode:
		c := v.buf[0]
		v.buf = v.buf[1:] // Remove the first byte for processing
		v.navigate(c)
	case CommandMode:
		hasEsc := v.HasEsc()
		if hasEsc >= 0 {
			v.cmdMode = NavMode          // Switch back to navigation mode on escape
			v.buf = v.buf[hasEsc+1:]     // Remove the escape sequence
			v.ap.MoveCursor(0, v.ap.H-1) // Move cursor to the command line
			v.ap.ClearEndOfLine()
			break
		}
		ret := bytes.IndexByte(v.buf, '\r')
		if ret < 0 {
			v.ap.WriteAt(0, v.ap.H-1, ":%s", string(v.buf))
		} else {
			data := v.buf[:ret]   // Get the command input up to the carriage return
			v.buf = v.buf[ret+1:] // Remove the command input from
			return v.command(data)
		}
	case InsertMode:
		// Handle insert mode input (e.g., add to buffer)
		str := string(v.buf) // probably ansiclean actually
		hasEsc := v.HasEsc()
		if hasEsc >= 0 {
			v.cmdMode = NavMode      // Switch back to navigation mode on escape
			str = str[:hasEsc]       // Get the string up to the escape character
			v.buf = v.buf[hasEsc+1:] // Remove the escape sequence
		} else {
			v.buf = nil
		}
		v.ap.WriteAt(v.cx, v.cy, str)
		v.cx += len(str) // Move cursor right by the length of the input
	}
	v.UpdateStatus()
	return true // Continue processing
}
