package main

import "testing"

func TestTintCursorPixelTransparent(t *testing.T) {
	// Transparent source leaves the background unchanged.
	if got := tintCursorPixel(0xff000000, 0xffffffff); got != 0 {
		t.Fatalf("transparent pixel=%#08x want 0", got)
	}
}

func TestTintCursorPixelOpaqueBlackBecomesDarkOutline(t *testing.T) {
	got := tintCursorPixel(0xff000000, 0xff000000)
	want := uint32(0xff000000 | cursorDarkRed<<16 | cursorDarkGreen<<8 | cursorDarkBlue)
	if got != want {
		t.Fatalf("black pixel=%#08x want %#08x", got, want)
	}
}

func TestTintCursorPixelOpaqueWhiteBecomesTaskbarTint(t *testing.T) {
	got := tintCursorPixel(0xffffffff, 0xffffffff)
	want := uint32(0xff000000 | tintRed<<16 | tintGreen<<8 | tintBlue)
	if got != want {
		t.Fatalf("white pixel=%#08x want %#08x", got, want)
	}
}

func TestTintCursorPixelHalfAlphaWhiteKeepsAlphaAndTint(t *testing.T) {
	// White source at 50% alpha renders ~127 on black and 255 on white.
	got := tintCursorPixel(0xff7f7f7f, 0xffffffff)
	alpha := int(got >> 24)
	if alpha < 126 || alpha > 128 {
		t.Fatalf("alpha=%d want ~127", alpha)
	}
	if r := int((got >> 16) & 0xff); r != tintRed {
		t.Fatalf("red=%d want %d", r, tintRed)
	}
	if g := int((got >> 8) & 0xff); g != tintGreen {
		t.Fatalf("green=%d want %d", g, tintGreen)
	}
	if b := int(got & 0xff); b != tintBlue {
		t.Fatalf("blue=%d want %d", b, tintBlue)
	}
}

func BenchmarkTintTypicalCursorSetPixels(b *testing.B) {
	const pixels = 64 * 64 * 2
	black := make([]uint32, pixels)
	white := make([]uint32, pixels)
	for i := 0; i < pixels; i++ {
		switch i % 4 {
		case 0:
			black[i], white[i] = 0xff000000, 0xffffffff // transparent
		case 1:
			black[i], white[i] = 0xff000000, 0xff000000 // black outline
		case 2:
			black[i], white[i] = 0xffffffff, 0xffffffff // white fill
		default:
			black[i], white[i] = 0xff7f7f7f, 0xffffffff // antialiased edge
		}
	}
	out := make([]uint32, pixels)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := 0; i < pixels; i++ {
			out[i] = tintCursorPixel(black[i], white[i])
		}
	}
	_ = out
}
