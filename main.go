package main

import (
	"flag"
	"os"

	"fortio.org/cli"
	"fortio.org/gvi/vi"
	"fortio.org/log"
	"fortio.org/terminal/ansipixels"
)

func main() {
	os.Exit(Main())
}

func Main() int {
	cli.MinArgs = 0
	cli.MaxArgs = 1
	cli.ArgsHelp = "[filename]\t\tto edit a file, vi style"
	cli.Main()
	ap := ansipixels.NewAnsiPixels(20.)
	err := ap.Open()
	if err != nil {
		return log.FErrf("Failed to open terminal: %v", err)
	}
	defer ap.Restore()
	vi := vi.NewVi(ap)
	ap.OnResize = vi.Update
	_ = ap.OnResize()
	if flag.NArg() == 1 {
		vi.Open(flag.Arg(0))
	}
	cont := true
	for cont {
		err = ap.ReadOrResizeOrSignal()
		if err != nil {
			return log.FErrf("Error reading terminal: %v", err)
		}
		if len(ap.Data) > 0 {
			cont = vi.Process()
		}
	}
	ap.MoveCursor(0, ap.H-1)
	return 0
}
