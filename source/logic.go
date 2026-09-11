package main

import "strings"

type Language uint8

const (
	LanguageUnknown Language = iota
	LanguageEnglish
	LanguageRussian
	LanguageOther
)

func (l Language) String() string {
	switch l {
	case LanguageEnglish:
		return "EN"
	case LanguageRussian:
		return "RU"
	case LanguageOther:
		return "OTHER"
	default:
		return "UNKNOWN"
	}
}

// Windows LANGID stores the primary language in the low 10 bits.
// English primary language id = 0x09, Russian = 0x19.
func languageFromHKL(hkl uintptr) Language {
	if hkl == 0 {
		return LanguageUnknown
	}
	langID := uint16(hkl & 0xffff)
	primary := langID & 0x03ff
	switch primary {
	case 0x09:
		return LanguageEnglish
	case 0x19:
		return LanguageRussian
	default:
		return LanguageOther
	}
}

func classifyAccessibleName(name string) Language {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return LanguageUnknown
	}
	if strings.Contains(n, "англий") || strings.Contains(n, "english") {
		return LanguageEnglish
	}
	if strings.Contains(n, "русск") || strings.Contains(n, "russian") {
		return LanguageRussian
	}
	return LanguageOther
}

const (
	vkShift    = 0x10
	vkControl  = 0x11
	vkMenu     = 0x12 // ALT
	vkSpace    = 0x20
	vkLWin     = 0x5b
	vkRWin     = 0x5c
	vkLShift   = 0xa0
	vkRShift   = 0xa1
	vkLControl = 0xa2
	vkRControl = 0xa3
	vkLMenu    = 0xa4
	vkRMenu    = 0xa5
)

type HotkeyDetector struct {
	shift   bool
	ctrl    bool
	alt     bool
	win     bool
	space   bool
	pending uint8
}

const (
	pendingNone uint8 = iota
	pendingAltShift
	pendingCtrlShift
	pendingWinSpace
)

func (d *HotkeyDetector) Handle(vk uint32, down bool) (layoutShortcut bool) {
	switch vk {
	case vkShift, vkLShift, vkRShift:
		d.shift = down
	case vkControl, vkLControl, vkRControl:
		d.ctrl = down
	case vkMenu, vkLMenu, vkRMenu:
		d.alt = down
	case vkLWin, vkRWin:
		d.win = down
	case vkSpace:
		d.space = down
	}

	if d.pending == pendingNone {
		switch {
		case d.alt && d.shift:
			d.pending = pendingAltShift
		case d.ctrl && d.shift:
			d.pending = pendingCtrlShift
		case d.win && d.space:
			d.pending = pendingWinSpace
		}
		return false
	}

	// Trigger only after BOTH keys of the detected chord have been released.
	// This makes the subsequent one-shot HKL read occur after Windows has had
	// the complete shortcut sequence, regardless of how long either key was held.
	switch d.pending {
	case pendingAltShift:
		if !d.alt && !d.shift {
			d.pending = pendingNone
			return true
		}
	case pendingCtrlShift:
		if !d.ctrl && !d.shift {
			d.pending = pendingNone
			return true
		}
	case pendingWinSpace:
		if !d.win && !d.space {
			d.pending = pendingNone
			return true
		}
	}
	return false
}

func shouldTint(lang Language, highContrast bool) bool {
	return lang == LanguageEnglish && !highContrast
}

type VisualGate struct {
	known bool
	last  bool
}

// Decide reports whether the expensive visual backend must run.
// Normal foreground/layout notifications with an unchanged RU/EN visual state
// are suppressed. force is reserved for Explorer/taskbar rebind events.
func (g *VisualGate) Decide(tint, force bool) bool {
	if force || !g.known || g.last != tint {
		g.known = true
		g.last = tint
		return true
	}
	return false
}

func (g *VisualGate) Reset() {
	g.known = false
}

func packABGR(a, r, g, b uint8) uint32 {
	return uint32(a)<<24 | uint32(b)<<16 | uint32(g)<<8 | uint32(r)
}

const (
	tintRed   = 183
	tintGreen = 233
	tintBlue  = 255
)
