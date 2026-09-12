//go:build windows

package main

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

type winPoint struct {
	X, Y int32
}

type winMsg struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       winPoint
	LPrivate uint32
}

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type highContrast struct {
	CbSize           uint32
	DwFlags          uint32
	DefaultSchemePtr uintptr
}

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type variant struct {
	VT        uint16
	Reserved1 uint16
	Reserved2 uint16
	Reserved3 uint16
	Val       int64
}

type filetime struct {
	LowDateTime  uint32
	HighDateTime uint32
}

type osVersionInfoEx struct {
	Size             uint32
	MajorVersion     uint32
	MinorVersion     uint32
	BuildNumber      uint32
	PlatformID       uint32
	CSDVersion       [128]uint16
	ServicePackMajor uint16
	ServicePackMinor uint16
	SuiteMask        uint16
	ProductType      byte
	Reserved         byte
}

// ACCENT_POLICY / WINDOWCOMPOSITIONATTRIBDATA are the structures used by
// SetWindowCompositionAttribute for the Windows 10 taskbar composition surface.
type accentPolicy struct {
	AccentState   int32
	AccentFlags   uint32
	GradientColor uint32
	AnimationID   int32
}

type windowCompositionAttribData struct {
	Attrib  int32
	Padding uint32
	Data    uintptr
	Size    uintptr
}

var iidIAccessible = guid{
	Data1: 0x618736E0,
	Data2: 0x3C3D,
	Data3: 0x11CF,
	Data4: [8]byte{0x81, 0x0C, 0x00, 0xAA, 0x00, 0x38, 0x9B, 0x71},
}

const (
	objIDClient = 0xFFFFFFFC
	vtI4        = 3

	wsPopup = 0x80000000

	wsExToolWindow = 0x00000080
	wsExNoActivate = 0x08000000

	wmDestroy               = 0x0002
	wmClose                 = 0x0010
	wmSettingChange         = 0x001A
	wmDisplayChange         = 0x007E
	wmTimer                 = 0x0113
	wmPowerBroadcast        = 0x0218
	wmThemeChanged          = 0x031A
	wmDwmCompositionChanged = 0x031E
	wmKeyDown               = 0x0100
	wmKeyUp                 = 0x0101
	wmSysKeyDown            = 0x0104
	wmSysKeyUp              = 0x0105
	wmApp                   = 0x8000

	spiGetHighContrast = 0x0042
	hcfHighContrastOn  = 0x00000001

	whKeyboardLL = 13
	hcAction     = 0

	eventSystemForeground  = 0x0003
	wineventOutOfContext   = 0x0000
	wineventSkipOwnProcess = 0x0002

	pbtApmResumeSuspend   = 0x0007
	pbtApmResumeAutomatic = 0x0012

	wcaAccentPolicy      = 19
	accentEnableGradient = 1 // Opaque colored taskbar surface.
	accentFlagsDefault   = 2

	smtoAbortIfHung = 0x0002
	smtoBlock       = 0x0001

	errorAlreadyExists = 183
	errorFileNotFound  = 2

	keySetValue = 0x0002
	regSZ       = 1

	processQueryLimitedInformation = 0x1000
	localeSLocalizedDisplayName    = 0x00000002
	localeSLocalizedLanguageName   = 0x0000006f
	localeSEnglishLanguageName     = 0x00001001
	localeSNativeLanguageName      = 0x00000004
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	oleacc   = syscall.NewLazyDLL("oleacc.dll")
	oleaut32 = syscall.NewLazyDLL("oleaut32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	ntdll    = syscall.NewLazyDLL("ntdll.dll")

	procFindWindowW                   = user32.NewProc("FindWindowW")
	procEnumWindows                   = user32.NewProc("EnumWindows")
	procEnumChildWindows              = user32.NewProc("EnumChildWindows")
	procGetClassNameW                 = user32.NewProc("GetClassNameW")
	procGetParent                     = user32.NewProc("GetParent")
	procIsWindow                      = user32.NewProc("IsWindow")
	procIsWindowVisible               = user32.NewProc("IsWindowVisible")
	procCreateWindowExW               = user32.NewProc("CreateWindowExW")
	procDestroyWindow                 = user32.NewProc("DestroyWindow")
	procRegisterClassExW              = user32.NewProc("RegisterClassExW")
	procUnregisterClassW              = user32.NewProc("UnregisterClassW")
	procDefWindowProcW                = user32.NewProc("DefWindowProcW")
	procRegisterWindowMessageW        = user32.NewProc("RegisterWindowMessageW")
	procSetTimer                      = user32.NewProc("SetTimer")
	procKillTimer                     = user32.NewProc("KillTimer")
	procGetMessageW                   = user32.NewProc("GetMessageW")
	procTranslateMessage              = user32.NewProc("TranslateMessage")
	procDispatchMessageW              = user32.NewProc("DispatchMessageW")
	procPostQuitMessage               = user32.NewProc("PostQuitMessage")
	procPostMessageW                  = user32.NewProc("PostMessageW")
	procGetForegroundWindow           = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId      = user32.NewProc("GetWindowThreadProcessId")
	procGetKeyboardLayout             = user32.NewProc("GetKeyboardLayout")
	procGetKeyboardLayoutList         = user32.NewProc("GetKeyboardLayoutList")
	procSetWindowsHookExW             = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx                = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx           = user32.NewProc("UnhookWindowsHookEx")
	procSetWinEventHook               = user32.NewProc("SetWinEventHook")
	procUnhookWinEvent                = user32.NewProc("UnhookWinEvent")
	procSystemParametersInfoW         = user32.NewProc("SystemParametersInfoW")
	procMessageBoxW                   = user32.NewProc("MessageBoxW")
	procSetWindowCompositionAttribute = user32.NewProc("SetWindowCompositionAttribute")
	procSendMessageTimeoutW           = user32.NewProc("SendMessageTimeoutW")

	procGetModuleHandleW           = kernel32.NewProc("GetModuleHandleW")
	procCreateEventW               = kernel32.NewProc("CreateEventW")
	procOpenEventW                 = kernel32.NewProc("OpenEventW")
	procSetEvent                   = kernel32.NewProc("SetEvent")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procWaitForSingleObject        = kernel32.NewProc("WaitForSingleObject")
	procCreateMutexW               = kernel32.NewProc("CreateMutexW")
	procOpenMutexW                 = kernel32.NewProc("OpenMutexW")
	procReleaseMutex               = kernel32.NewProc("ReleaseMutex")
	procGetCurrentProcess          = kernel32.NewProc("GetCurrentProcess")
	procGetProcessTimes            = kernel32.NewProc("GetProcessTimes")
	procGetWindowsDirectoryW       = kernel32.NewProc("GetWindowsDirectoryW")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procLCIDToLocaleName           = kernel32.NewProc("LCIDToLocaleName")
	procGetLocaleInfoEx            = kernel32.NewProc("GetLocaleInfoEx")

	procOleInitialize   = ole32.NewProc("OleInitialize")
	procOleUninitialize = ole32.NewProc("OleUninitialize")

	procAccessibleObjectFromWindow = oleacc.NewProc("AccessibleObjectFromWindow")
	procSysStringLen               = oleaut32.NewProc("SysStringLen")
	procSysFreeString              = oleaut32.NewProc("SysFreeString")

	procRegSetValueExW  = advapi32.NewProc("RegSetValueExW")
	procRegDeleteValueW = advapi32.NewProc("RegDeleteValueW")

	procRtlMoveMemory = ntdll.NewProc("RtlMoveMemory")
	procRtlGetVersion = ntdll.NewProc("RtlGetVersion")
)

func utf16Ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		panic(err)
	}
	return p
}

func boolCall(r uintptr, name string, callErr error) error {
	if r != 0 {
		return nil
	}
	if callErr != nil && callErr != syscall.Errno(0) {
		return fmt.Errorf("%s: %w", name, callErr)
	}
	return fmt.Errorf("%s failed", name)
}

func validateWindowsABI() error {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		return fmt.Errorf("unsupported pointer size: %d; Windows x64 required", unsafe.Sizeof(uintptr(0)))
	}
	if unsafe.Sizeof(winMsg{}) != 48 {
		return fmt.Errorf("MSG ABI size=%d, want 48", unsafe.Sizeof(winMsg{}))
	}
	if unsafe.Sizeof(wndClassEx{}) != 80 {
		return fmt.Errorf("WNDCLASSEX ABI size=%d, want 80", unsafe.Sizeof(wndClassEx{}))
	}
	if unsafe.Sizeof(accentPolicy{}) != 16 {
		return fmt.Errorf("ACCENT_POLICY ABI size=%d, want 16", unsafe.Sizeof(accentPolicy{}))
	}
	if unsafe.Sizeof(windowCompositionAttribData{}) != 24 {
		return fmt.Errorf("WINDOWCOMPOSITIONATTRIBDATA ABI size=%d, want 24", unsafe.Sizeof(windowCompositionAttribData{}))
	}
	if unsafe.Sizeof(osVersionInfoEx{}) != 284 {
		return fmt.Errorf("OSVERSIONINFOEXW ABI size=%d, want 284", unsafe.Sizeof(osVersionInfoEx{}))
	}
	return nil
}

func windowsVersion() (major, minor, build uint32, err error) {
	vi := osVersionInfoEx{Size: uint32(unsafe.Sizeof(osVersionInfoEx{}))}
	r, _, _ := procRtlGetVersion.Call(uintptr(unsafe.Pointer(&vi)))
	if int32(uint32(r)) < 0 {
		return 0, 0, 0, fmt.Errorf("RtlGetVersion NTSTATUS=0x%08X", uint32(r))
	}
	return vi.MajorVersion, vi.MinorVersion, vi.BuildNumber, nil
}

func validateSupportedWindows() error {
	major, minor, build, err := windowsVersion()
	if err != nil {
		return err
	}
	support := windowsBuildSupport(major, minor, build)
	if !support.Supported {
		return fmt.Errorf("unsupported Windows %d.%d build %d: %s", major, minor, build, support.Reason)
	}
	return nil
}

func showInfoMessage(title, text string) {
	const mbOK = 0x00000000
	const mbIconInformation = 0x00000040
	procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr(text))),
		uintptr(unsafe.Pointer(utf16Ptr(title))),
		mbOK|mbIconInformation,
	)
}

func getModuleHandle() uintptr {
	r, _, _ := procGetModuleHandleW.Call(0)
	return r
}

func findWindow(className string) uintptr {
	r, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(utf16Ptr(className))), 0)
	return r
}

func isWindow(hwnd uintptr) bool {
	r, _, _ := procIsWindow.Call(hwnd)
	return r != 0
}

func isWindowVisible(hwnd uintptr) bool {
	r, _, _ := procIsWindowVisible.Call(hwnd)
	return r != 0
}

func getParent(hwnd uintptr) uintptr {
	r, _, _ := procGetParent.Call(hwnd)
	return r
}

func getClassName(hwnd uintptr) string {
	buf := make([]uint16, 128)
	r, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:r])
}

func enumerateTaskbarsRaw() []uintptr {
	var result []uintptr
	cb := syscall.NewCallback(func(hwnd, lparam uintptr) uintptr {
		cls := getClassName(hwnd)
		if cls == "Shell_TrayWnd" || cls == "Shell_SecondaryTrayWnd" {
			result = append(result, hwnd)
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	runtime.KeepAlive(cb)
	return result
}

func windowProcessImagePath(hwnd uintptr) (string, error) {
	var pid uint32
	tid, _, _ := procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if tid == 0 || pid == 0 {
		return "", errors.New("GetWindowThreadProcessId returned no process")
	}
	h, _, e := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if h == 0 {
		return "", fmt.Errorf("OpenProcess(pid=%d): %w", pid, e)
	}
	defer closeHandle(h)
	buf := make([]uint16, 32768)
	sz := uint32(len(buf))
	r, _, e := procQueryFullProcessImageNameW.Call(
		h,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&sz)),
	)
	if r == 0 || sz == 0 {
		return "", fmt.Errorf("QueryFullProcessImageNameW(pid=%d): %w", pid, e)
	}
	return syscall.UTF16ToString(buf[:sz]), nil
}

func getWindowsDirectory() (string, error) {
	buf := make([]uint16, 32768)
	r, _, e := procGetWindowsDirectoryW.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r == 0 || int(r) >= len(buf) {
		return "", fmt.Errorf("GetWindowsDirectoryW: %w", e)
	}
	return syscall.UTF16ToString(buf[:r]), nil
}

func isVerifiedExplorerTaskbar(hwnd uintptr) bool {
	if hwnd == 0 || !isWindow(hwnd) {
		return false
	}
	cls := getClassName(hwnd)
	if cls != "Shell_TrayWnd" && cls != "Shell_SecondaryTrayWnd" {
		return false
	}
	path, err := windowProcessImagePath(hwnd)
	if err != nil {
		return false
	}
	windowsDir, err := getWindowsDirectory()
	return err == nil && isExpectedExplorerProcessPath(path, windowsDir)
}

func enumerateTaskbars() []uintptr {
	raw := enumerateTaskbarsRaw()
	result := make([]uintptr, 0, len(raw))
	for _, hwnd := range raw {
		if isVerifiedExplorerTaskbar(hwnd) {
			result = append(result, hwnd)
		}
	}
	return result
}

func findInputIndicatorButton() uintptr {
	var taskbar uintptr
	for _, candidate := range enumerateTaskbars() {
		if getClassName(candidate) == "Shell_TrayWnd" {
			taskbar = candidate
			break
		}
	}
	if taskbar == 0 {
		return 0
	}
	var found uintptr
	cb := syscall.NewCallback(func(hwnd, lparam uintptr) uintptr {
		if getClassName(hwnd) != "InputIndicatorButton" {
			return 1
		}
		p := getParent(hwnd)
		if p != 0 && getClassName(p) == "TrayInputIndicatorWClass" {
			found = hwnd
			return 0
		}
		return 1
	})
	procEnumChildWindows.Call(taskbar, cb, 0)
	runtime.KeepAlive(cb)
	return found
}

func oleInitialize() error {
	r, _, _ := procOleInitialize.Call(0)
	hr := int32(uint32(r))
	if hr < 0 {
		return fmt.Errorf("OleInitialize HRESULT=0x%08X", uint32(hr))
	}
	return nil
}

func oleUninitialize() { procOleUninitialize.Call() }

func accessibleName(hwnd uintptr) (string, error) {
	var object unsafe.Pointer
	r, _, _ := procAccessibleObjectFromWindow.Call(
		hwnd,
		uintptr(uint32(objIDClient)),
		uintptr(unsafe.Pointer(&iidIAccessible)),
		uintptr(unsafe.Pointer(&object)),
	)
	hr := int32(uint32(r))
	if hr < 0 || object == nil {
		return "", fmt.Errorf("AccessibleObjectFromWindow HRESULT=0x%08X", uint32(hr))
	}

	vtblPtr := *(*unsafe.Pointer)(object)
	if vtblPtr == nil {
		return "", errors.New("IAccessible vtable is null")
	}
	vtbl := (*[28]uintptr)(vtblPtr)
	defer syscall.SyscallN(vtbl[2], uintptr(object))

	child := variant{VT: vtI4, Val: 0}
	var bstr *uint16
	r, _, _ = syscall.SyscallN(
		vtbl[10],
		uintptr(object),
		uintptr(unsafe.Pointer(&child)),
		uintptr(unsafe.Pointer(&bstr)),
	)
	hr = int32(uint32(r))
	if hr < 0 {
		return "", fmt.Errorf("IAccessible.get_accName HRESULT=0x%08X", uint32(hr))
	}
	if bstr == nil {
		return "", errors.New("IAccessible.get_accName returned null BSTR")
	}
	defer procSysFreeString.Call(uintptr(unsafe.Pointer(bstr)))

	n, _, _ := procSysStringLen.Call(uintptr(unsafe.Pointer(bstr)))
	if n == 0 {
		return "", nil
	}
	u16 := unsafe.Slice(bstr, int(n))
	return syscall.UTF16ToString(u16), nil
}

func localeNameFromLangID(langID uint16) (string, error) {
	buf := make([]uint16, 85)
	r, _, e := procLCIDToLocaleName.Call(
		uintptr(uint32(langID)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0,
	)
	if r == 0 {
		return "", fmt.Errorf("LCIDToLocaleName(langid=0x%04X): %w", langID, e)
	}
	return syscall.UTF16ToString(buf), nil
}

func localeInfoString(localeName string, infoType uint32) (string, error) {
	need, _, e := procGetLocaleInfoEx.Call(
		uintptr(unsafe.Pointer(utf16Ptr(localeName))),
		uintptr(infoType),
		0,
		0,
	)
	if need == 0 {
		return "", fmt.Errorf("GetLocaleInfoEx(%s,%#x) size: %w", localeName, infoType, e)
	}
	buf := make([]uint16, int(need))
	r, _, e := procGetLocaleInfoEx.Call(
		uintptr(unsafe.Pointer(utf16Ptr(localeName))),
		uintptr(infoType),
		uintptr(unsafe.Pointer(&buf[0])),
		need,
	)
	if r == 0 {
		return "", fmt.Errorf("GetLocaleInfoEx(%s,%#x): %w", localeName, infoType, e)
	}
	return syscall.UTF16ToString(buf), nil
}

func installedKeyboardLayouts() ([]uintptr, error) {
	count, _, e := procGetKeyboardLayoutList.Call(0, 0)
	if count == 0 {
		return nil, fmt.Errorf("GetKeyboardLayoutList(count): %w", e)
	}
	layouts := make([]uintptr, int(count))
	got, _, e := procGetKeyboardLayoutList.Call(count, uintptr(unsafe.Pointer(&layouts[0])))
	if got == 0 {
		return nil, fmt.Errorf("GetKeyboardLayoutList(data): %w", e)
	}
	return layouts[:int(got)], nil
}

func installedLanguageNameCandidates() ([]languageNameCandidate, error) {
	layouts, err := installedKeyboardLayouts()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var out []languageNameCandidate
	for _, hkl := range layouts {
		lang := languageFromHKL(hkl)
		if lang == LanguageUnknown {
			continue
		}
		langID := uint16(hkl & 0xffff)
		localeName, err := localeNameFromLangID(langID)
		if err != nil {
			continue
		}
		for _, typ := range []uint32{
			localeSLocalizedDisplayName,
			localeSLocalizedLanguageName,
			localeSEnglishLanguageName,
			localeSNativeLanguageName,
		} {
			label, err := localeInfoString(localeName, typ)
			if err != nil || normalizeLanguageLabel(label) == "" {
				continue
			}
			key := fmt.Sprintf("%d:%s", lang, normalizeLanguageLabel(label))
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, languageNameCandidate{Name: label, Language: lang})
		}
	}
	if len(out) == 0 {
		return nil, errors.New("Windows NLS returned no installed language names")
	}
	return out, nil
}

func readMSAALanguage() (Language, string, error) {
	hwnd := findInputIndicatorButton()
	if hwnd == 0 {
		return LanguageUnknown, "", errors.New("InputIndicatorButton not found")
	}
	name, err := accessibleName(hwnd)
	if err != nil {
		return LanguageUnknown, "", err
	}
	if candidates, candidateErr := installedLanguageNameCandidates(); candidateErr == nil {
		if lang := classifyAccessibleNameWithCandidates(name, candidates); lang != LanguageUnknown {
			return lang, name, nil
		}
	}
	// Last-resort compatibility for systems where NLS enumeration unexpectedly
	// fails. This is not the primary localization path.
	lang := classifyAccessibleName(name)
	if lang == LanguageUnknown {
		return lang, name, errors.New("MSAA name unclassified")
	}
	return lang, name, nil
}

func foregroundLanguage() (Language, uintptr, uint32, error) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return LanguageUnknown, 0, 0, errors.New("GetForegroundWindow returned NULL")
	}
	tid, _, _ := procGetWindowThreadProcessId.Call(hwnd, 0)
	if tid == 0 {
		return LanguageUnknown, hwnd, 0, errors.New("GetWindowThreadProcessId returned 0")
	}
	hkl, _, _ := procGetKeyboardLayout.Call(tid)
	if hkl == 0 {
		return LanguageUnknown, hwnd, uint32(tid), errors.New("GetKeyboardLayout returned 0")
	}
	return languageFromHKL(hkl), hwnd, uint32(tid), nil
}

func queryHighContrast() (bool, error) {
	hc := highContrast{CbSize: uint32(unsafe.Sizeof(highContrast{}))}
	r, _, e := procSystemParametersInfoW.Call(
		spiGetHighContrast,
		uintptr(hc.CbSize),
		uintptr(unsafe.Pointer(&hc)),
		0,
	)
	if err := boolCall(r, "SystemParametersInfoW(SPI_GETHIGHCONTRAST)", e); err != nil {
		return true, err
	}
	return hc.DwFlags&hcfHighContrastOn != 0, nil
}

func registerWindowClass(className string, wndProc, brush uintptr) (uint16, error) {
	wc := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		LpfnWndProc:   wndProc,
		HInstance:     getModuleHandle(),
		HbrBackground: brush,
		LpszClassName: utf16Ptr(className),
	}
	r, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if r == 0 {
		return 0, fmt.Errorf("RegisterClassExW(%s): %w", className, e)
	}
	return uint16(r), nil
}

func unregisterWindowClass(className string) {
	procUnregisterClassW.Call(uintptr(unsafe.Pointer(utf16Ptr(className))), getModuleHandle())
}

func createControlWindow(className string) (uintptr, error) {
	r, _, e := procCreateWindowExW.Call(
		wsExToolWindow|wsExNoActivate,
		uintptr(unsafe.Pointer(utf16Ptr(className))),
		0,
		wsPopup,
		0, 0, 0, 0,
		0, 0, getModuleHandle(), 0,
	)
	if r == 0 {
		return 0, fmt.Errorf("CreateWindowExW(control): %w", e)
	}
	return r, nil
}

func destroyWindow(hwnd uintptr) {
	if hwnd != 0 {
		procDestroyWindow.Call(hwnd)
	}
}

func defWindowProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return r
}

func registerWindowMessage(name string) uint32 {
	r, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(utf16Ptr(name))))
	return uint32(r)
}

func setTimer(hwnd uintptr, id uintptr, ms uint32) error {
	r, _, e := procSetTimer.Call(hwnd, id, uintptr(ms), 0)
	if r == 0 {
		return fmt.Errorf("SetTimer(%d): %w", id, e)
	}
	return nil
}

func killTimer(hwnd uintptr, id uintptr) {
	procKillTimer.Call(hwnd, id)
}

func postMessage(hwnd uintptr, msg uint32, wparam, lparam uintptr) bool {
	r, _, _ := procPostMessageW.Call(hwnd, uintptr(msg), wparam, lparam)
	return r != 0
}

func messageLoop() error {
	var msg winMsg
	for {
		r, _, e := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		signed := int32(r)
		if signed == 0 {
			return nil
		}
		if signed == -1 {
			return fmt.Errorf("GetMessageW: %w", e)
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func postQuitMessage(code int32) { procPostQuitMessage.Call(uintptr(uint32(code))) }

func installKeyboardHook(callback uintptr) (uintptr, error) {
	h, _, e := procSetWindowsHookExW.Call(whKeyboardLL, callback, getModuleHandle(), 0)
	if h == 0 {
		return 0, fmt.Errorf("SetWindowsHookExW(WH_KEYBOARD_LL): %w", e)
	}
	return h, nil
}

func callNextKeyboardHook(hook, nCode, wparam, lparam uintptr) uintptr {
	r, _, _ := procCallNextHookEx.Call(hook, nCode, wparam, lparam)
	return r
}

func unhookKeyboard(h uintptr) {
	if h != 0 {
		procUnhookWindowsHookEx.Call(h)
	}
}

func copyVKCodeFromHook(lparam uintptr) uint32 {
	var vk uint32
	procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&vk)), lparam, unsafe.Sizeof(vk))
	return vk
}

func installForegroundHook(callback uintptr) (uintptr, error) {
	h, _, e := procSetWinEventHook.Call(
		eventSystemForeground,
		eventSystemForeground,
		0,
		callback,
		0, 0,
		wineventOutOfContext|wineventSkipOwnProcess,
	)
	if h == 0 {
		return 0, fmt.Errorf("SetWinEventHook(EVENT_SYSTEM_FOREGROUND): %w", e)
	}
	return h, nil
}

func unhookWinEvent(h uintptr) {
	if h != 0 {
		procUnhookWinEvent.Call(h)
	}
}

func compositionAPIAvailable() error {
	if err := procSetWindowCompositionAttribute.Find(); err != nil {
		return fmt.Errorf("SetWindowCompositionAttribute unavailable: %w", err)
	}
	if err := procSendMessageTimeoutW.Find(); err != nil {
		return fmt.Errorf("SendMessageTimeoutW unavailable: %w", err)
	}
	return nil
}

func setTaskbarOpaqueTint(hwnd uintptr, r, g, b uint8) error {
	if hwnd == 0 || !isWindow(hwnd) {
		return errors.New("taskbar HWND is invalid")
	}
	cls := getClassName(hwnd)
	if cls != "Shell_TrayWnd" && cls != "Shell_SecondaryTrayWnd" {
		return fmt.Errorf("unexpected taskbar class %q", cls)
	}
	if err := compositionAPIAvailable(); err != nil {
		return err
	}
	policy := accentPolicy{
		AccentState:   accentEnableGradient,
		AccentFlags:   accentFlagsDefault,
		GradientColor: packABGR(0xFF, r, g, b),
		AnimationID:   0,
	}
	data := windowCompositionAttribData{
		Attrib: wcaAccentPolicy,
		Data:   uintptr(unsafe.Pointer(&policy)),
		Size:   unsafe.Sizeof(policy),
	}
	ok, _, callErr := procSetWindowCompositionAttribute.Call(
		hwnd,
		uintptr(unsafe.Pointer(&data)),
	)
	runtime.KeepAlive(policy)
	runtime.KeepAlive(data)
	if ok == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return fmt.Errorf("SetWindowCompositionAttribute: %w", callErr)
		}
		return errors.New("SetWindowCompositionAttribute returned FALSE")
	}
	return nil
}

func restoreTaskbarNormal(hwnd uintptr) error {
	if hwnd == 0 || !isWindow(hwnd) {
		return nil
	}
	if err := compositionAPIAvailable(); err != nil {
		return err
	}
	var result uintptr
	ok, _, callErr := procSendMessageTimeoutW.Call(
		hwnd,
		wmDwmCompositionChanged,
		1,
		0,
		smtoAbortIfHung|smtoBlock,
		250,
		uintptr(unsafe.Pointer(&result)),
	)
	if ok == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return fmt.Errorf("SendMessageTimeoutW(WM_DWMCOMPOSITIONCHANGED): %w", callErr)
		}
		return errors.New("SendMessageTimeoutW(WM_DWMCOMPOSITIONCHANGED) failed")
	}
	return nil
}

func createNamedEvent(name string, manualReset, initialState bool) (uintptr, bool, error) {
	manual := uintptr(0)
	if manualReset {
		manual = 1
	}
	initial := uintptr(0)
	if initialState {
		initial = 1
	}
	r, _, e := procCreateEventW.Call(0, manual, initial, uintptr(unsafe.Pointer(utf16Ptr(name))))
	if r == 0 {
		return 0, false, fmt.Errorf("CreateEventW(%s): %w", name, e)
	}
	return r, e == syscall.Errno(errorAlreadyExists), nil
}

func openNamedEvent(name string, access uint32) uintptr {
	r, _, _ := procOpenEventW.Call(uintptr(access), 0, uintptr(unsafe.Pointer(utf16Ptr(name))))
	return r
}

func setEvent(h uintptr) bool {
	r, _, _ := procSetEvent.Call(h)
	return r != 0
}

func closeHandle(h uintptr) {
	if h != 0 {
		procCloseHandle.Call(h)
	}
}

func waitForSingleObject(h uintptr, timeoutMS uint32) uint32 {
	r, _, _ := procWaitForSingleObject.Call(h, uintptr(timeoutMS))
	return uint32(r)
}

func createSingleInstanceMutex(name string) (uintptr, bool, error) {
	r, _, e := procCreateMutexW.Call(0, 1, uintptr(unsafe.Pointer(utf16Ptr(name))))
	if r == 0 {
		return 0, false, fmt.Errorf("CreateMutexW: %w", e)
	}
	if e == syscall.Errno(errorAlreadyExists) {
		closeHandle(r)
		return 0, true, nil
	}
	return r, false, nil
}

func releaseMutex(h uintptr) {
	if h != 0 {
		procReleaseMutex.Call(h)
		closeHandle(h)
	}
}

func isMutexPresent(name string) bool {
	const synchronize = 0x00100000
	r, _, _ := procOpenMutexW.Call(synchronize, 0, uintptr(unsafe.Pointer(utf16Ptr(name))))
	if r == 0 {
		return false
	}
	closeHandle(r)
	return true
}

func processCPU100ns() (uint64, error) {
	process, _, _ := procGetCurrentProcess.Call()
	var creation, exit, kernel, user filetime
	r, _, e := procGetProcessTimes.Call(
		process,
		uintptr(unsafe.Pointer(&creation)),
		uintptr(unsafe.Pointer(&exit)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r == 0 {
		return 0, fmt.Errorf("GetProcessTimes: %w", e)
	}
	to64 := func(ft filetime) uint64 {
		return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
	}
	return to64(kernel) + to64(user), nil
}

func setRunValue(name, command string) error {
	var key syscall.Handle
	sub := utf16Ptr(`Software\Microsoft\Windows\CurrentVersion\Run`)
	if err := syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, sub, 0, syscall.KEY_SET_VALUE, &key); err != nil {
		return fmt.Errorf("RegOpenKeyEx Run: %w", err)
	}
	defer syscall.RegCloseKey(key)

	value := syscall.StringToUTF16(command)
	r, _, e := procRegSetValueExW.Call(
		uintptr(key),
		uintptr(unsafe.Pointer(utf16Ptr(name))),
		0,
		regSZ,
		uintptr(unsafe.Pointer(&value[0])),
		uintptr(len(value)*2),
	)
	if r != 0 {
		return fmt.Errorf("RegSetValueExW code=%d err=%v", r, e)
	}
	return nil
}

func deleteRunValue(name string) error {
	var key syscall.Handle
	sub := utf16Ptr(`Software\Microsoft\Windows\CurrentVersion\Run`)
	if err := syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, sub, 0, syscall.KEY_SET_VALUE, &key); err != nil {
		return fmt.Errorf("RegOpenKeyEx Run: %w", err)
	}
	defer syscall.RegCloseKey(key)
	r, _, e := procRegDeleteValueW.Call(uintptr(key), uintptr(unsafe.Pointer(utf16Ptr(name))))
	if r == 0 || uint32(r) == errorFileNotFound {
		return nil
	}
	return fmt.Errorf("RegDeleteValueW code=%d err=%v", r, e)
}
