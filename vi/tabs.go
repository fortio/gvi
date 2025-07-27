package vi

import "fortio.org/log"

func (v *Vi) UpdateTabs() []int {
	v.ap.WriteString("\r\t")
	var tabs []int
	prevX := 0
	for {
		x, _, err := v.ap.ReadCursorPosXY()
		if err != nil {
			log.Errf("Error reading cursor position: %v", err)
			return nil
		}
		if x == prevX || x == v.ap.W-1 {
			break
		}
		tabs = append(tabs, x)
		v.ap.WriteString("\t")
		prevX = x
	}
	return tabs
}
