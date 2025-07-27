package vi

import "fortio.org/log"

func (v *Vi) UpdateTabs() {
	v.ap.WriteString("\r\t")
	v.tabs = v.tabs[:0]
	prevX := 0
	for {
		x, _, err := v.ap.ReadCursorPosXY()
		if err != nil {
			log.Errf("Error reading cursor position: %v", err)
			return
		}
		if x == prevX || x == v.ap.W-1 {
			break
		}
		v.tabs = append(v.tabs, x)
		v.ap.WriteString("\t")
		prevX = x
	}
}
