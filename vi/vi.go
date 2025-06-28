package vi // import "fortio.org/gvi/vi"

import "fortio.org/terminal/ansipixels"

type Vi struct {
	cmdMode  bool
	ap       *ansipixels.AnsiPixels
	filename string // Not used in this example, but could be used to track the file being edited
}

func NewVi(ap *ansipixels.AnsiPixels) *Vi {
	return &Vi{
		cmdMode:  true,
		ap:       ap,
		filename: "...", // Placeholder for filename
	}
}

func (v *Vi) ToggleMode() {
	if v.cmdMode {
		v.cmdMode = false
	} else {
		v.cmdMode = true
	}
}

func (v *Vi) Update() error {
	v.ap.ClearScreen()
	v.UpdateStatus()
	v.ap.WriteBoxed(v.ap.H/2, "Hello, World!\nHiya caches\n'q' to quit\nEsc or I switch mode\ntry resize\n")
	return nil
}

func (v *Vi) UpdateStatus() {
	mode := "Command"
	if !v.cmdMode {
		mode = "Insert"
	}
	v.ap.WriteAt(0, v.ap.H-1, "%s File: %s - Mode: %s - %dx%d %s",
		ansipixels.Inverse, v.filename, mode, v.ap.W, v.ap.H,
		ansipixels.Reset)
}

func (v *Vi) Process(data []byte) bool {
	if len(data) == 0 {
		return true // No input, continue
	}

	if v.cmdMode {
		switch data[0] {
		case 'i': // Enter insert mode
			v.ToggleMode()
		case 'q': // Quit
			return false
		default:
			// Handle other command mode inputs
			return true
		}
	} else {
		switch data[0] {
		case 27: // Escape to command mode
			v.ToggleMode()
		default:
			// Handle insert mode inputs (e.g., add to buffer)
		}
	}
	v.UpdateStatus()
	return true // Continue processing
}
