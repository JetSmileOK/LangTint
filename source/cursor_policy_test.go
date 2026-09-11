package main

import "testing"

func TestTintPolicyOnlyArrowAndHand(t *testing.T) {
	want := map[uint32]bool{oCrNormal: true, oCrHand: true}
	if len(tintedSystemCursorIDs) != 2 {
		t.Fatalf("tintedSystemCursorIDs len=%d; want 2", len(tintedSystemCursorIDs))
	}
	seen := map[uint32]bool{}
	for _, id := range tintedSystemCursorIDs {
		if !want[id] {
			t.Fatalf("unexpected tinted cursor id=%d", id)
		}
		if seen[id] {
			t.Fatalf("duplicate tinted cursor id=%d", id)
		}
		seen[id] = true
	}
	for id := range want {
		if !seen[id] {
			t.Fatalf("required tinted cursor id=%d missing", id)
		}
	}
}

func TestTextAndUtilityCursorsAreNotTinted(t *testing.T) {
	excluded := []uint32{
		oCrIBeam, oCrWait, oCrCross, oCrUp,
		oCrSizeNWSE, oCrSizeNESW, oCrSizeWE, oCrSizeNS,
		oCrSizeAll, oCrNo, oCrAppStarting,
	}
	for _, excludedID := range excluded {
		for _, tintedID := range tintedSystemCursorIDs {
			if excludedID == tintedID {
				t.Fatalf("cursor id=%d must remain in the Windows scheme", excludedID)
			}
		}
	}
}
