package imgtmpl

import (
	"fmt"
	"image/color"
	"strconv"
)

var (
	enableLog = false
)

func EnableLog() {
	enableLog = true
}

func parseColor(text string) (c color.Color, err error) {
	var v uint64
	switch len(text) {
	case 8:
		if v, err = strconv.ParseUint(text, 16, 64); err == nil {
			c = color.RGBA{
				R: uint8(v >> 24),
				G: uint8((v >> 16) & 0xFF),
				B: uint8((v >> 8) & 0xFF),
				A: uint8(v & 0xFF),
			}
		}
	case 6:
		if v, err = strconv.ParseUint(text, 16, 64); err == nil {
			c = color.RGBA{
				R: uint8(v >> 16),
				G: uint8((v >> 8) & 0xFF),
				B: uint8(v & 0xFF),
				A: 255,
			}
		}
	default:
		err = fmt.Errorf("未知色彩格式: %s", text)
	}
	return
}
