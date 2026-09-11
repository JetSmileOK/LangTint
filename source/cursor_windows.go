//go:build windows

package main

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

type iconInfo struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

type bitmapObject struct {
	Type       int32
	Width      int32
	Height     int32
	WidthBytes int32
	Planes     uint16
	BitsPixel  uint16
	Bits       uintptr
}

type bitmapV5Header struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
	RedMask       uint32
	GreenMask     uint32
	BlueMask      uint32
	AlphaMask     uint32
	CSType        uint32
	Endpoints     [36]byte
	GammaRed      uint32
	GammaGreen    uint32
	GammaBlue     uint32
	Intent        uint32
	ProfileData   uint32
	ProfileSize   uint32
	Reserved      uint32
}

var (
	cursorUser32 = syscall.NewLazyDLL("user32.dll")
	cursorGdi32  = syscall.NewLazyDLL("gdi32.dll")

	procLoadCursorW        = cursorUser32.NewProc("LoadCursorW")
	procGetIconInfo        = cursorUser32.NewProc("GetIconInfo")
	procDrawIconEx         = cursorUser32.NewProc("DrawIconEx")
	procCreateIconIndirect = cursorUser32.NewProc("CreateIconIndirect")
	procDestroyCursor      = cursorUser32.NewProc("DestroyCursor")
	procSetSystemCursor    = cursorUser32.NewProc("SetSystemCursor")

	procCreateCompatibleDC = cursorGdi32.NewProc("CreateCompatibleDC")
	procDeleteDC           = cursorGdi32.NewProc("DeleteDC")
	procCreateDIBSection   = cursorGdi32.NewProc("CreateDIBSection")
	procSelectObject       = cursorGdi32.NewProc("SelectObject")
	procDeleteGDIObject    = cursorGdi32.NewProc("DeleteObject")
	procCreateBitmap       = cursorGdi32.NewProc("CreateBitmap")
	procGetObjectW         = cursorGdi32.NewProc("GetObjectW")
)

const (
	spiSetCursors = 0x0057

	imageCursor = 2
	diNormal    = 0x0003
	diNoMirror  = 0x0010

	biBitFields  = 3
	dibRGBColors = 0
	lcsSRGB      = 0x73524742
)

func validateCursorABI() error {
	if unsafe.Sizeof(iconInfo{}) != 32 {
		return fmt.Errorf("ICONINFO ABI size=%d, want 32", unsafe.Sizeof(iconInfo{}))
	}
	if unsafe.Sizeof(bitmapObject{}) != 32 {
		return fmt.Errorf("BITMAP ABI size=%d, want 32", unsafe.Sizeof(bitmapObject{}))
	}
	if unsafe.Sizeof(bitmapV5Header{}) != 124 {
		return fmt.Errorf("BITMAPV5HEADER ABI size=%d, want 124", unsafe.Sizeof(bitmapV5Header{}))
	}
	return nil
}

func cursorAPIAvailable() error {
	for name, proc := range map[string]*syscall.LazyProc{
		"LoadCursorW":        procLoadCursorW,
		"GetIconInfo":        procGetIconInfo,
		"DrawIconEx":         procDrawIconEx,
		"CreateIconIndirect": procCreateIconIndirect,
		"SetSystemCursor":    procSetSystemCursor,
		"CreateDIBSection":   procCreateDIBSection,
	} {
		if err := proc.Find(); err != nil {
			return fmt.Errorf("%s unavailable: %w", name, err)
		}
	}
	return validateCursorABI()
}

func restoreSystemCursors() error {
	r, _, e := procSystemParametersInfoW.Call(spiSetCursors, 0, 0, 0)
	return boolCall(r, "SystemParametersInfoW(SPI_SETCURSORS)", e)
}

func getBitmapObject(hbmp uintptr) (bitmapObject, error) {
	var bm bitmapObject
	r, _, e := procGetObjectW.Call(
		hbmp,
		unsafe.Sizeof(bm),
		uintptr(unsafe.Pointer(&bm)),
	)
	if r == 0 {
		return bm, fmt.Errorf("GetObjectW(bitmap): %w", e)
	}
	return bm, nil
}

func cursorGeometry(hcur uintptr) (width, height int, hotspotX, hotspotY uint32, err error) {
	var ii iconInfo
	r, _, e := procGetIconInfo.Call(hcur, uintptr(unsafe.Pointer(&ii)))
	if r == 0 {
		return 0, 0, 0, 0, fmt.Errorf("GetIconInfo: %w", e)
	}
	defer func() {
		if ii.HbmColor != 0 {
			procDeleteGDIObject.Call(ii.HbmColor)
		}
		if ii.HbmMask != 0 {
			procDeleteGDIObject.Call(ii.HbmMask)
		}
	}()

	target := ii.HbmColor
	monochrome := false
	if target == 0 {
		target = ii.HbmMask
		monochrome = true
	}
	if target == 0 {
		return 0, 0, 0, 0, errors.New("cursor has no color or mask bitmap")
	}
	bm, err := getBitmapObject(target)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	w := int(bm.Width)
	h := int(bm.Height)
	if w < 0 {
		w = -w
	}
	if h < 0 {
		h = -h
	}
	if monochrome {
		h /= 2
	}
	if w <= 0 || h <= 0 || w > 512 || h > 512 {
		return 0, 0, 0, 0, fmt.Errorf("invalid cursor geometry %dx%d", w, h)
	}
	return w, h, ii.XHotspot, ii.YHotspot, nil
}

func makeV5Header(width, height int) bitmapV5Header {
	return bitmapV5Header{
		Size:        uint32(unsafe.Sizeof(bitmapV5Header{})),
		Width:       int32(width),
		Height:      -int32(height), // top-down DIB
		Planes:      1,
		BitCount:    32,
		Compression: biBitFields,
		RedMask:     0x00ff0000,
		GreenMask:   0x0000ff00,
		BlueMask:    0x000000ff,
		AlphaMask:   0xff000000,
		CSType:      lcsSRGB,
	}
}

func createDIB32(width, height int) (hbmp uintptr, bits unsafe.Pointer, err error) {
	if width <= 0 || height <= 0 {
		return 0, nil, errors.New("invalid DIB geometry")
	}
	hdr := makeV5Header(width, height)
	var p unsafe.Pointer
	h, _, e := procCreateDIBSection.Call(
		0,
		uintptr(unsafe.Pointer(&hdr)),
		dibRGBColors,
		uintptr(unsafe.Pointer(&p)),
		0,
		0,
	)
	runtime.KeepAlive(hdr)
	if h == 0 || p == nil {
		return 0, nil, fmt.Errorf("CreateDIBSection: %w", e)
	}
	return h, p, nil
}

func renderCursorOnBackground(hcur uintptr, width, height int, bg uint32) ([]uint32, error) {
	dc, _, e := procCreateCompatibleDC.Call(0)
	if dc == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC: %w", e)
	}
	defer procDeleteDC.Call(dc)

	hbmp, bits, err := createDIB32(width, height)
	if err != nil {
		return nil, err
	}
	defer procDeleteGDIObject.Call(hbmp)

	old, _, e := procSelectObject.Call(dc, hbmp)
	if old == 0 || old == ^uintptr(0) {
		return nil, fmt.Errorf("SelectObject(DIB): %w", e)
	}
	defer procSelectObject.Call(dc, old)

	pix := unsafe.Slice((*uint32)(bits), width*height)
	for i := range pix {
		pix[i] = bg
	}

	r, _, e := procDrawIconEx.Call(
		dc,
		0,
		0,
		hcur,
		uintptr(width),
		uintptr(height),
		0,
		0,
		diNormal|diNoMirror,
	)
	if r == 0 {
		return nil, fmt.Errorf("DrawIconEx: %w", e)
	}
	out := make([]uint32, len(pix))
	copy(out, pix)
	return out, nil
}

func createTintedCursorFromSystemID(id uint32) (uintptr, error) {
	hcur, _, e := procLoadCursorW.Call(0, uintptr(id))
	if hcur == 0 {
		return 0, fmt.Errorf("LoadCursorW(%d): %w", id, e)
	}
	w, h, hotX, hotY, err := cursorGeometry(hcur)
	if err != nil {
		return 0, fmt.Errorf("cursor %d geometry: %w", id, err)
	}

	black, err := renderCursorOnBackground(hcur, w, h, 0xff000000)
	if err != nil {
		return 0, fmt.Errorf("cursor %d black render: %w", id, err)
	}
	white, err := renderCursorOnBackground(hcur, w, h, 0xffffffff)
	if err != nil {
		return 0, fmt.Errorf("cursor %d white render: %w", id, err)
	}

	colorBitmap, bits, err := createDIB32(w, h)
	if err != nil {
		return 0, fmt.Errorf("cursor %d output DIB: %w", id, err)
	}
	defer procDeleteGDIObject.Call(colorBitmap)
	out := unsafe.Slice((*uint32)(bits), w*h)
	for i := range out {
		out[i] = tintCursorPixel(black[i], white[i])
	}

	rowBytes := ((w + 15) / 16) * 2
	maskBytes := make([]byte, rowBytes*h)
	var maskPtr uintptr
	if len(maskBytes) != 0 {
		maskPtr = uintptr(unsafe.Pointer(&maskBytes[0]))
	}
	maskBitmap, _, e := procCreateBitmap.Call(
		uintptr(w),
		uintptr(h),
		1,
		1,
		maskPtr,
	)
	if maskBitmap == 0 {
		return 0, fmt.Errorf("CreateBitmap(mask cursor %d): %w", id, e)
	}
	defer procDeleteGDIObject.Call(maskBitmap)

	ii := iconInfo{
		FIcon:    0,
		XHotspot: hotX,
		YHotspot: hotY,
		HbmMask:  maskBitmap,
		HbmColor: colorBitmap,
	}
	created, _, e := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	runtime.KeepAlive(ii)
	runtime.KeepAlive(maskBytes)
	if created == 0 {
		return 0, fmt.Errorf("CreateIconIndirect(cursor %d): %w", id, e)
	}
	return created, nil
}

// applyTintedSystemCursors replaces only the normal Arrow and Hand roles with
// recoloured copies of the user's current scheme. All text/editing/status cursor
// roles remain untouched. On any failure it immediately reloads the user's
// normal cursor scheme so a partial replacement cannot stay.
func applyTintedSystemCursors() (int, error) {
	if err := cursorAPIAvailable(); err != nil {
		return 0, err
	}
	// Always source shapes from the user's configured scheme, not from a cursor
	// that this process may previously have tinted.
	if err := restoreSystemCursors(); err != nil {
		return 0, fmt.Errorf("pre-tint cursor restore: %w", err)
	}

	setCount := 0
	for _, id := range tintedSystemCursorIDs {
		hcur, err := createTintedCursorFromSystemID(id)
		if err != nil {
			_ = restoreSystemCursors()
			return setCount, err
		}
		r, _, callErr := procSetSystemCursor.Call(hcur, uintptr(id))
		if r == 0 {
			// SetSystemCursor destroys hcur on success only. On failure we own it.
			procDestroyCursor.Call(hcur)
			_ = restoreSystemCursors()
			return setCount, fmt.Errorf("SetSystemCursor(%d): %w", id, callErr)
		}
		setCount++
	}
	return setCount, nil
}
