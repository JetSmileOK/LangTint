package main

const (
	cursorDarkRed   = 20
	cursorDarkGreen = 47
	cursorDarkBlue  = 66
)

func channel(pixel uint32, shift uint) int {
	return int((pixel >> shift) & 0xff)
}

func clampByte(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// tintCursorPixel reconstructs approximate source alpha/luminance by comparing
// the same cursor pixel rendered over black and white backgrounds. This works
// for both alpha cursors and classic AND/XOR cursors without reading private
// cursor bitmap formats directly.
//
// Pixels are represented as 0xAARRGGBB, matching a 32-bit BGRA DIB in memory on
// little-endian Windows.
func tintCursorPixel(overBlack, overWhite uint32) uint32 {
	br := channel(overBlack, 16)
	bg := channel(overBlack, 8)
	bb := channel(overBlack, 0)
	wr := channel(overWhite, 16)
	wg := channel(overWhite, 8)
	wb := channel(overWhite, 0)

	// For normal alpha compositing: white-black = 255*(1-alpha).
	dr := clampByte(wr - br)
	dg := clampByte(wg - bg)
	db := clampByte(wb - bb)
	diff := (dr + dg + db) / 3
	alpha := 255 - diff
	if alpha < 8 {
		return 0
	}

	// Recover the original source colour from the black-background render.
	sr := clampByte((br*255 + alpha/2) / alpha)
	sg := clampByte((bg*255 + alpha/2) / alpha)
	sb := clampByte((bb*255 + alpha/2) / alpha)
	luma := (54*sr + 183*sg + 19*sb + 128) >> 8

	// Keep dark parts dark for a crisp outline. Bright parts become exactly the
	// taskbar tint. Intermediate antialiasing shades are smoothly interpolated.
	t := 0
	if luma > 96 {
		t = (luma - 96) * 255 / (255 - 96)
		if t > 255 {
			t = 255
		}
	}

	mix := func(lo, hi int) int {
		return lo + (hi-lo)*t/255
	}
	r := mix(cursorDarkRed, tintRed)
	g := mix(cursorDarkGreen, tintGreen)
	b := mix(cursorDarkBlue, tintBlue)

	return uint32(alpha)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)
}
