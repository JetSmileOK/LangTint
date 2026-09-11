package main

import (
	"math/rand"
	"testing"
)

func TestLanguageFromHKL(t *testing.T) {
	cases := []struct {
		hkl  uintptr
		want Language
	}{
		{0, LanguageUnknown},
		{0x0409, LanguageEnglish},
		{0x0809, LanguageEnglish}, // UK
		{0x0419, LanguageRussian},
		{0x0411, LanguageOther},
		{0x0000000004090409, LanguageEnglish},
	}
	for _, tc := range cases {
		if got := languageFromHKL(tc.hkl); got != tc.want {
			t.Fatalf("languageFromHKL(%#x)=%v want %v", tc.hkl, got, tc.want)
		}
	}
}

func TestAccessibleFallback(t *testing.T) {
	cases := []struct {
		name string
		want Language
	}{
		{"Индикатор ввода в области уведомлений - Английский (США)", LanguageEnglish},
		{"Индикатор ввода в области уведомлений - Русский", LanguageRussian},
		{"Input indicator - English (United States)", LanguageEnglish},
		{"Input indicator - Russian", LanguageRussian},
		{"Input indicator - German", LanguageOther},
		{"", LanguageUnknown},
	}
	for _, tc := range cases {
		if got := classifyAccessibleName(tc.name); got != tc.want {
			t.Fatalf("classifyAccessibleName(%q)=%v want %v", tc.name, got, tc.want)
		}
	}
}

func feed(d *HotkeyDetector, seq ...struct {
	vk   uint32
	down bool
}) int {
	count := 0
	for _, e := range seq {
		if d.Handle(e.vk, e.down) {
			count++
		}
	}
	return count
}

func kd(v uint32) struct {
	vk   uint32
	down bool
} {
	return struct {
		vk   uint32
		down bool
	}{v, true}
}
func ku(v uint32) struct {
	vk   uint32
	down bool
} {
	return struct {
		vk   uint32
		down bool
	}{v, false}
}

func TestAltShiftBothOrdersExactlyOnce(t *testing.T) {
	for _, seq := range [][]struct {
		vk   uint32
		down bool
	}{
		{kd(vkLMenu), kd(vkLShift), ku(vkLShift), ku(vkLMenu)},
		{kd(vkLShift), kd(vkLMenu), ku(vkLMenu), ku(vkLShift)},
	} {
		d := &HotkeyDetector{}
		if got := feed(d, seq...); got != 1 {
			t.Fatalf("got %d shortcut events, want 1", got)
		}
	}
}

func TestShortcutTriggersOnReleaseNotPress(t *testing.T) {
	d := &HotkeyDetector{}
	if d.Handle(vkLMenu, true) {
		t.Fatal("Alt down must not trigger")
	}
	if d.Handle(vkLShift, true) {
		t.Fatal("Alt+Shift press must not trigger before release")
	}
	for i := 0; i < 20; i++ {
		if d.Handle(vkLShift, true) {
			t.Fatal("held/repeated Shift must not trigger")
		}
	}
	if d.Handle(vkLShift, false) {
		t.Fatal("first modifier release must not trigger while Alt is still held")
	}
	if !d.Handle(vkLMenu, false) {
		t.Fatal("releasing both keys must trigger exactly once")
	}
}

func TestNoRepeatWhileHeld(t *testing.T) {
	d := &HotkeyDetector{}
	seq := []struct {
		vk   uint32
		down bool
	}{
		kd(vkLMenu), kd(vkLShift), kd(vkLShift), kd(vkLShift),
		ku(vkLShift), ku(vkLMenu),
	}
	if got := feed(d, seq...); got != 1 {
		t.Fatalf("got %d shortcut events, want 1", got)
	}
}

func TestRepeatedAltShift(t *testing.T) {
	d := &HotkeyDetector{}
	count := 0
	for i := 0; i < 1000; i++ {
		count += feed(d, kd(vkLMenu), kd(vkLShift), ku(vkLShift), ku(vkLMenu))
	}
	if count != 1000 {
		t.Fatalf("got %d shortcut events, want 1000", count)
	}
}

func TestCtrlShiftAndWinSpace(t *testing.T) {
	d := &HotkeyDetector{}
	if got := feed(d, kd(vkLControl), kd(vkLShift), ku(vkLShift), ku(vkLControl)); got != 1 {
		t.Fatalf("Ctrl+Shift got %d", got)
	}
	if got := feed(d, kd(vkLWin), kd(vkSpace), ku(vkSpace), ku(vkLWin)); got != 1 {
		t.Fatalf("Win+Space got %d", got)
	}
}

func TestOrdinaryTypingAndAltGrDoNotTrigger(t *testing.T) {
	d := &HotkeyDetector{}
	seq := []struct {
		vk   uint32
		down bool
	}{
		kd('A'), ku('A'), kd('B'), ku('B'),
		kd(vkRMenu), kd(vkLControl), kd('Q'), ku('Q'), ku(vkLControl), ku(vkRMenu),
	}
	if got := feed(d, seq...); got != 0 {
		t.Fatalf("false positive count=%d", got)
	}
}

func TestRandomKeyNoiseOnlyShortcutPatternsTrigger(t *testing.T) {
	for seed := int64(1); seed <= 500; seed++ {
		r := rand.New(rand.NewSource(seed))
		d := &HotkeyDetector{}
		for i := 0; i < 1000; i++ {
			vk := uint32(0x30 + r.Intn(0x5a-0x30+1))
			if d.Handle(vk, true) {
				t.Fatalf("seed=%d step=%d false trigger on key down %#x", seed, i, vk)
			}
			if d.Handle(vk, false) {
				t.Fatalf("seed=%d step=%d false trigger on key up %#x", seed, i, vk)
			}
		}
	}
}

func TestTintPolicy(t *testing.T) {
	if !shouldTint(LanguageEnglish, false) {
		t.Fatal("EN should tint")
	}
	for _, tc := range []struct {
		lang Language
		hc   bool
	}{
		{LanguageRussian, false},
		{LanguageOther, false},
		{LanguageUnknown, false},
		{LanguageEnglish, true},
	} {
		if shouldTint(tc.lang, tc.hc) {
			t.Fatalf("unexpected tint: %+v", tc)
		}
	}
}

func TestPackABGROpaquePaleBlue(t *testing.T) {
	got := packABGR(0xFF, 183, 233, 255)
	const want uint32 = 0xFFFFE9B7
	if got != want {
		t.Fatalf("packABGR=%#08x want %#08x", got, want)
	}
}

func TestVisualGateSuppressesUnchangedState(t *testing.T) {
	var g VisualGate
	if !g.Decide(false, false) {
		t.Fatal("initial state must be applied")
	}
	for i := 0; i < 1000; i++ {
		if g.Decide(false, false) {
			t.Fatalf("unchanged RU state triggered at iteration %d", i)
		}
	}
	if !g.Decide(true, false) {
		t.Fatal("RU->EN must trigger")
	}
	for i := 0; i < 1000; i++ {
		if g.Decide(true, false) {
			t.Fatalf("unchanged EN state triggered at iteration %d", i)
		}
	}
	if !g.Decide(false, false) {
		t.Fatal("EN->RU must trigger")
	}
}

func TestVisualGateForceRebind(t *testing.T) {
	var g VisualGate
	if !g.Decide(true, false) {
		t.Fatal("initial EN must trigger")
	}
	if g.Decide(true, false) {
		t.Fatal("unchanged EN must be suppressed")
	}
	if !g.Decide(true, true) {
		t.Fatal("forced Explorer rebind must trigger even if EN is unchanged")
	}
	if g.Decide(true, false) {
		t.Fatal("state after forced rebind must again be suppressed")
	}
	g.Reset()
	if !g.Decide(true, false) {
		t.Fatal("reset must make next visual state observable")
	}
}
