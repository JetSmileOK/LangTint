package main

const (
	oCrNormal      = 32512
	oCrIBeam       = 32513
	oCrWait        = 32514
	oCrCross       = 32515
	oCrUp          = 32516
	oCrSizeNWSE    = 32642
	oCrSizeNESW    = 32643
	oCrSizeWE      = 32644
	oCrSizeNS      = 32645
	oCrSizeAll     = 32646
	oCrNo          = 32648
	oCrHand        = 32649
	oCrAppStarting = 32650
)

// Only visually substantial pointer shapes are tinted.
// Text/editing and status cursors deliberately remain in the user's normal
// Windows cursor scheme so they keep maximum contrast on application surfaces.
var tintedSystemCursorIDs = [...]uint32{
	oCrNormal,
	oCrHand,
}
