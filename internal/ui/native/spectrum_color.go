package native

import (
	"image/color"
	"math"
)

// spectrumRGB follows a continuous hue wheel with softened saturation and
// constant HSV value. Time and frequency supply the phase; audio is unchanged.
func spectrumRGB(hue float64) color.NRGBA {
	hue -= math.Floor(hue)
	h := hue * 6
	x := 1 - math.Abs(math.Mod(h, 2)-1)
	var r, g, b float64
	switch int(h) {
	case 0:
		r, g = 1, x
	case 1:
		r, g = x, 1
	case 2:
		g, b = 1, x
	case 3:
		g, b = x, 1
	case 4:
		r, b = x, 1
	default:
		r, b = 1, x
	}
	channel := func(v float64) uint8 { return uint8(math.Round((.24 + .66*v) * 255)) }
	return color.NRGBA{R: channel(r), G: channel(g), B: channel(b), A: 255}
}
