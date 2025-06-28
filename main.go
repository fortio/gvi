package main

import (
	"os"

	"fortio.org/cli"
	"fortio.org/log"
	"fortio.org/terminal/ansipixels"
)

func main() {
	os.Exit(Main())
}

func Main() int {
	cli.Main()
	ap := ansipixels.NewAnsiPixels(20.)
	err := ap.Open()
	if err != nil {
		return log.FErrf("Failed to open terminal: %v", err)
	}
	defer ap.Restore()
	ap.ClearScreen()
	ap.WriteBoxed(ap.H/2, "Hello, World!\nHiya caches")
	_ = ap.ReadOrResizeOrSignal()
	ap.MoveCursor(0, ap.H-1)
	return 0
}
