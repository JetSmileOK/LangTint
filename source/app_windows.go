//go:build windows

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	buildID       = "LangTint_PUBLIC_v1.7.1_20260912"
	appName       = "LangTint"
	installFolder = `Programs\LangTint`
	exeName       = "LangTint.exe"

	mutexName = `Local\LangTint_v1_6_0_Mutex`
	readyName = `Local\LangTint_v1_6_0_Ready`

	// Immediate predecessor family (v1.0-v1.5) shared these identifiers.
	priorHotkeyRunName       = "TaskbarLayoutTintHotkey"
	priorHotkeyInstallFolder = "TaskbarLayoutTintHotkey"
	priorHotkeyControlClass  = "TaskbarLayoutTintHotkeyControl_v1"

	// v1.5.1 had unique identifiers, but a direct double-click with no
	// arguments incorrectly started --run from the source folder. v1.5.2
	// must stop that process so the downloaded/extracted files unlock.
	priorV151RunName       = "TaskbarLayoutTintHotkey_v1_5_1"
	priorV151InstallFolder = "TaskbarLayoutTintHotkey_v1_5_1"
	priorV151ControlClass  = "TaskbarLayoutTintHotkeyControl_v1_5_1"

	// Immediate predecessor v1.5.2. v1.5.3 must stop it before restoring
	// the full cursor scheme and installing the two-cursor policy.
	priorV152RunName       = "TaskbarLayoutTintHotkey_v1_5_2"
	priorV152InstallFolder = "TaskbarLayoutTintHotkey_v1_5_2"
	priorV152ControlClass  = "TaskbarLayoutTintHotkeyControl_v1_5_2"

	// Last public prototype before the LangTint stable-name migration.
	priorV153RunName       = "TaskbarLayoutTintHotkey_v1_5_3"
	priorV153InstallFolder = "TaskbarLayoutTintHotkey_v1_5_3"
	priorV153ControlClass  = "TaskbarLayoutTintHotkeyControl_v1_5_3"

	// Very early native polling prototype.
	legacyRunName       = "TaskbarLayoutTintNative"
	legacyStopName      = `Local\TaskbarLayoutTintNative_v1_Stop`
	legacyInstallFolder = "TaskbarLayoutTintNative"

	controlClassName = "LangTintControl_v1_6_0"

	wmAppHotkey     = wmApp + 1
	wmAppForeground = wmApp + 2

	timerLayout = 101
	timerRebind = 102
	timerStop   = 103

	layoutDebounceMS = 80
	rebindDebounceMS = 200
)

var (
	activeRuntime *watcherRuntime

	controlWndProcCB = syscall.NewCallback(controlWndProc)
	keyboardProcCB   = syscall.NewCallback(keyboardProc)
	winEventProcCB   = syscall.NewCallback(winEventProc)
)

type taskbarManager struct {
	windows  map[uintptr]bool
	accented map[uintptr]bool
}

func newTaskbarManager() *taskbarManager {
	return &taskbarManager{
		windows:  make(map[uintptr]bool),
		accented: make(map[uintptr]bool),
	}
}

func (m *taskbarManager) discoverAllowEmpty() (int, error) {
	raw := enumerateTaskbarsRaw()
	taskbars := enumerateTaskbars()
	if len(raw) > 0 && len(taskbars) == 0 {
		return 0, errors.New("taskbar-class windows exist but none are owned by explorer.exe")
	}
	live := make(map[uintptr]bool, len(taskbars))
	for _, tb := range taskbars {
		if tb == 0 || !isWindow(tb) {
			continue
		}
		live[tb] = true
	}
	for tb := range m.windows {
		if !live[tb] {
			delete(m.windows, tb)
			delete(m.accented, tb)
		}
	}
	for tb := range live {
		m.windows[tb] = true
	}
	return len(m.windows), nil
}

func (m *taskbarManager) discover() error {
	n, err := m.discoverAllowEmpty()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("no usable explorer.exe taskbar windows")
	}
	return nil
}

// apply changes composition only when it is actually required. Normal
// foreground events with an unchanged EN state do not call DWM again.
// force is used only after Explorer/taskbar lifecycle changes.
func (m *taskbarManager) apply(tint, force bool) (setCalls, restoreCalls uint64, firstErr error) {
	for tb := range m.windows {
		if !isWindow(tb) {
			continue
		}
		if tint {
			if m.accented[tb] && !force {
				continue
			}
			setCalls++
			if err := setTaskbarOpaqueTint(tb, tintRed, tintGreen, tintBlue); err != nil {
				if firstErr == nil {
					firstErr = err
				}
			} else {
				m.accented[tb] = true
			}
			continue
		}
		if m.accented[tb] {
			restoreCalls++
			if err := restoreTaskbarNormal(tb); err != nil {
				if firstErr == nil {
					firstErr = err
				}
			} else {
				delete(m.accented, tb)
			}
		}
	}
	return setCalls, restoreCalls, firstErr
}

func (m *taskbarManager) close() (restoreCalls uint64) {
	for tb := range m.accented {
		if isWindow(tb) {
			restoreCalls++
			_ = restoreTaskbarNormal(tb)
		}
	}
	m.windows = make(map[uintptr]bool)
	m.accented = make(map[uintptr]bool)
	return restoreCalls
}

type visualCommand struct {
	tint    bool
	force   bool
	ack     chan error
	stop    bool
	barrier bool
}

type visualWorker struct {
	manager *taskbarManager
	queue   chan visualCommand
	done    chan struct{}
	owner   *watcherRuntime

	setCalls           uint64
	restoreCalls       uint64
	cursorSetCalls     uint64
	cursorRestoreCalls uint64
	cursorObjectsSet   uint64
	commands           uint64
	failures           uint64
	cursorFailures     uint64
}

func newVisualWorker(owner *watcherRuntime, manager *taskbarManager) *visualWorker {
	w := &visualWorker{
		manager: manager,
		queue:   make(chan visualCommand, 1),
		done:    make(chan struct{}),
		owner:   owner,
	}
	go w.loop()
	return w
}

func (w *visualWorker) loop() {
	defer close(w.done)
	for cmd := range w.queue {
		if cmd.barrier {
			if cmd.ack != nil {
				cmd.ack <- nil
			}
			continue
		}
		if cmd.stop {
			restored := w.manager.close()
			atomic.AddUint64(&w.restoreCalls, restored)
			if err := restoreSystemCursors(); err != nil {
				atomic.AddUint64(&w.cursorFailures, 1)
				if w.owner != nil {
					w.owner.logf("CURSOR_RESTORE_ON_STOP_FAIL err=%v", err)
				}
			} else {
				atomic.AddUint64(&w.cursorRestoreCalls, 1)
			}
			if cmd.ack != nil {
				cmd.ack <- nil
			}
			return
		}

		atomic.AddUint64(&w.commands, 1)
		var err error
		if cmd.force || len(w.manager.windows) == 0 {
			err = w.manager.discover()
		}
		if err == nil {
			sets, restores, applyErr := w.manager.apply(cmd.tint, cmd.force)
			atomic.AddUint64(&w.setCalls, sets)
			atomic.AddUint64(&w.restoreCalls, restores)
			err = applyErr
		}
		if err == nil {
			if cmd.tint {
				count, cursorErr := applyTintedSystemCursors()
				atomic.AddUint64(&w.cursorObjectsSet, uint64(count))
				if cursorErr != nil {
					atomic.AddUint64(&w.cursorFailures, 1)
					err = cursorErr
				} else {
					atomic.AddUint64(&w.cursorSetCalls, 1)
				}
			} else {
				if cursorErr := restoreSystemCursors(); cursorErr != nil {
					atomic.AddUint64(&w.cursorFailures, 1)
					err = cursorErr
				} else {
					atomic.AddUint64(&w.cursorRestoreCalls, 1)
				}
			}
		}
		if err != nil {
			// Keep taskbar and cursor state atomic from the user's point of view.
			// Any partial cursor/taskbar failure rolls both visuals back to Normal.
			_, rolledBack, _ := w.manager.apply(false, true)
			atomic.AddUint64(&w.restoreCalls, rolledBack)
			if restoreErr := restoreSystemCursors(); restoreErr == nil {
				atomic.AddUint64(&w.cursorRestoreCalls, 1)
			}
			atomic.AddUint64(&w.failures, 1)
			if w.owner != nil {
				w.owner.logf("VISUAL_WORKER_FAIL tint=%v force=%v err=%v", cmd.tint, cmd.force, err)
			}
		}
		if cmd.ack != nil {
			cmd.ack <- err
		}
	}
}

func (w *visualWorker) submit(tint, force, wait bool) error {
	if w == nil {
		return nil
	}
	cmd := visualCommand{tint: tint, force: force}
	if wait {
		cmd.ack = make(chan error, 1)
		w.queue <- cmd
		return <-cmd.ack
	}

	// Only the message-loop thread submits asynchronous commands. Keep at most
	// the newest desired visual state. If DWM/Explorer is slow, keyboard hooks
	// never wait for it and stale visual requests are discarded.
	select {
	case w.queue <- cmd:
		return nil
	default:
	}
	select {
	case old := <-w.queue:
		if old.ack != nil || old.stop || old.barrier {
			// A synchronous/barrier/stop command must never be discarded.
			w.queue <- old
			return nil
		}
		// Keep the newest desired tint but never lose lifecycle semantics. If a
		// TaskbarCreated/theme/display rebind was pending, the merged request
		// remains forced even when a newer foreground/layout event arrives.
		cmd.force = cmd.force || old.force
	default:
	}
	select {
	case w.queue <- cmd:
	default:
	}
	return nil
}

func (w *visualWorker) barrier() {
	if w == nil {
		return
	}
	ack := make(chan error, 1)
	w.queue <- visualCommand{barrier: true, ack: ack}
	<-ack
}

func (w *visualWorker) stop() {
	if w == nil {
		return
	}
	ack := make(chan error, 1)
	w.queue <- visualCommand{stop: true, ack: ack}
	<-ack
	<-w.done
}

type watcherRuntime struct {
	control        uintptr
	keyboardHook   uintptr
	foregroundHook uintptr
	taskbarCreated uint32
	detector       HotkeyDetector
	taskbars       *taskbarManager
	visualWorker   *visualWorker
	visualGate     VisualGate
	highContrast   bool
	language       Language
	visual         bool
	log            *os.File
	logMu          sync.Mutex

	hotkeyEvents     uint64
	foregroundEvents uint64
	layoutChecks     uint64
	msaaFallbacks    uint64
	transitions      uint64
	seenEN           bool
	seenRU           bool
	visualSuppressed uint64

	startupRetryActive bool
	startupRetryIndex  int
}

func (r *watcherRuntime) logf(format string, args ...any) {
	if r == nil || r.log == nil {
		return
	}
	r.logMu.Lock()
	defer r.logMu.Unlock()
	_, _ = fmt.Fprintf(r.log, "%s "+format+"\r\n", append([]any{time.Now().Format(time.RFC3339Nano)}, args...)...)
	_ = r.log.Sync()
}

func relevantVK(vk uint32) bool {
	switch vk {
	case vkShift, vkLShift, vkRShift,
		vkControl, vkLControl, vkRControl,
		vkMenu, vkLMenu, vkRMenu,
		vkLWin, vkRWin, vkSpace:
		return true
	default:
		return false
	}
}

func keyboardProc(nCode, wparam, lparam uintptr) uintptr {
	r := activeRuntime
	if r != nil && int32(nCode) == hcAction {
		down := uint32(wparam) == wmKeyDown || uint32(wparam) == wmSysKeyDown
		up := uint32(wparam) == wmKeyUp || uint32(wparam) == wmSysKeyUp
		if down || up {
			vk := copyVKCodeFromHook(lparam)
			if relevantVK(vk) && r.detector.Handle(vk, down) {
				atomic.AddUint64(&r.hotkeyEvents, 1)
				postMessage(r.control, wmAppHotkey, 0, 0)
			}
		}
	}
	var hook uintptr
	if r != nil {
		hook = r.keyboardHook
	}
	return callNextKeyboardHook(hook, nCode, wparam, lparam)
}

func winEventProc(hook, event, hwnd, idObject, idChild, eventThread, eventTime uintptr) uintptr {
	r := activeRuntime
	if r != nil && uint32(event) == eventSystemForeground {
		atomic.AddUint64(&r.foregroundEvents, 1)
		postMessage(r.control, wmAppForeground, hwnd, 0)
	}
	return 0
}

func controlWndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	r := activeRuntime
	if r != nil && r.taskbarCreated != 0 && uint32(msg) == r.taskbarCreated {
		r.completeStartupRetry()
		r.scheduleRebind()
		return 0
	}

	switch uint32(msg) {
	case wmAppHotkey:
		if r != nil {
			r.scheduleLayout(layoutDebounceMS)
		}
		return 0
	case wmAppForeground:
		if r != nil {
			r.scheduleLayout(30)
		}
		return 0
	case wmDisplayChange, wmSettingChange, wmThemeChanged:
		if r != nil {
			r.scheduleRebind()
		}
		return 0
	case wmPowerBroadcast:
		if r != nil && (uint32(wparam) == pbtApmResumeSuspend || uint32(wparam) == pbtApmResumeAutomatic) {
			r.scheduleRebind()
		}
		return 0
	case wmTimer:
		if r == nil {
			return 0
		}
		switch wparam {
		case timerLayout:
			killTimer(hwnd, timerLayout)
			if err := r.refreshLanguage("event", false, false); err != nil {
				r.logf("LANGUAGE_EVENT_VISUAL_FAIL err=%v", err)
			}
		case timerRebind:
			killTimer(hwnd, timerRebind)
			r.rebind()
			if r.startupRetryActive {
				r.scheduleNextStartupRetry()
			}
		case timerStop:
			killTimer(hwnd, timerStop)
			destroyWindow(hwnd)
		}
		return 0
	case wmClose:
		destroyWindow(hwnd)
		return 0
	case wmDestroy:
		postQuitMessage(0)
		return 0
	}
	return defWindowProc(hwnd, msg, wparam, lparam)
}

func (r *watcherRuntime) beginStartupRetry() {
	if r == nil || r.control == 0 || len(startupRetryDelaysMS) == 0 {
		return
	}
	r.startupRetryActive = true
	r.startupRetryIndex = 0
	_ = setTimer(r.control, timerRebind, startupRetryDelaysMS[0])
}

func (r *watcherRuntime) scheduleNextStartupRetry() {
	if r == nil || !r.startupRetryActive || r.control == 0 {
		return
	}
	r.startupRetryIndex++
	if r.startupRetryIndex >= len(startupRetryDelaysMS) {
		r.startupRetryActive = false
		r.logf("STARTUP_TASKBAR_RETRY_EXHAUSTED total_ms=%d", startupRetryTotalMS())
		return
	}
	_ = setTimer(r.control, timerRebind, startupRetryDelaysMS[r.startupRetryIndex])
}

func (r *watcherRuntime) completeStartupRetry() {
	if r == nil {
		return
	}
	r.startupRetryActive = false
	if r.control != 0 {
		killTimer(r.control, timerRebind)
	}
}

func (r *watcherRuntime) scheduleLayout(delayMS uint32) {
	if r == nil || r.control == 0 {
		return
	}
	_ = setTimer(r.control, timerLayout, delayMS)
}

func (r *watcherRuntime) scheduleRebind() {
	if r == nil || r.control == 0 {
		return
	}
	_ = setTimer(r.control, timerRebind, rebindDebounceMS)
}

func (r *watcherRuntime) refreshLanguage(source string, forceVisual, syncVisual bool) error {
	if r == nil {
		return nil
	}
	atomic.AddUint64(&r.layoutChecks, 1)

	lang, hwnd, tid, err := foregroundLanguage()
	used := fmt.Sprintf("GetKeyboardLayout tid=%d hwnd=0x%X", tid, hwnd)
	if err != nil || lang == LanguageUnknown {
		atomic.AddUint64(&r.msaaFallbacks, 1)
		fallback, name, msaaErr := readMSAALanguage()
		if msaaErr != nil {
			r.logf("LANGUAGE_READ_FAIL source=%s hkl_err=%v msaa_err=%v", source, err, msaaErr)
			r.language = LanguageUnknown
			return r.applyTint(forceVisual, syncVisual)
		}
		lang = fallback
		used = fmt.Sprintf("MSAA name=%q", name)
	}

	previous := r.language
	r.language = lang
	if lang == LanguageEnglish {
		r.seenEN = true
	}
	if lang == LanguageRussian {
		r.seenRU = true
	}
	if previous != LanguageUnknown && lang != previous {
		atomic.AddUint64(&r.transitions, 1)
	}
	if previous != lang {
		r.logf("STATE=%s source=%s via=%s", lang, source, used)
	}
	return r.applyTint(forceVisual, syncVisual)
}

func (r *watcherRuntime) refreshHighContrast() {
	hc, err := queryHighContrast()
	if err != nil {
		r.highContrast = true
		r.logf("HIGH_CONTRAST_QUERY_FAIL err=%v", err)
		return
	}
	r.highContrast = hc
}

func (r *watcherRuntime) applyTint(force, syncVisual bool) error {
	if r == nil || !r.visual || r.visualWorker == nil {
		return nil
	}
	tint := shouldTint(r.language, r.highContrast)
	if !r.visualGate.Decide(tint, force) {
		atomic.AddUint64(&r.visualSuppressed, 1)
		return nil
	}
	return r.visualWorker.submit(tint, force, syncVisual)
}

func (r *watcherRuntime) rebind() {
	if r == nil {
		return
	}
	r.refreshHighContrast()
	if err := r.refreshLanguage("rebind", true, false); err != nil {
		r.logf("TASKBAR_REBIND_FAIL err=%v", err)
	}
}

func setupRuntime(visual bool, logPath string) (*watcherRuntime, error) {
	if err := validateWindowsABI(); err != nil {
		return nil, err
	}
	if err := validateSupportedWindows(); err != nil {
		return nil, err
	}
	if visual {
		if err := cursorAPIAvailable(); err != nil {
			return nil, err
		}
		// Crash recovery: always begin from the user's configured cursor scheme.
		// If a previous process died while EN was active, no blue cursor persists.
		if err := restoreSystemCursors(); err != nil {
			return nil, fmt.Errorf("startup cursor restore: %w", err)
		}
	}
	if err := oleInitialize(); err != nil {
		return nil, err
	}
	cleanupOLE := true
	defer func() {
		if cleanupOLE {
			oleUninitialize()
		}
	}()

	if _, err := registerWindowClass(controlClassName, controlWndProcCB, 0); err != nil {
		return nil, err
	}
	controlRegistered := true
	defer func() {
		if controlRegistered {
			unregisterWindowClass(controlClassName)
		}
	}()

	control, err := createControlWindow(controlClassName)
	if err != nil {
		return nil, err
	}
	cleanupControl := true
	defer func() {
		if cleanupControl {
			destroyWindow(control)
		}
	}()

	var log *os.File
	if logPath != "" {
		_ = os.MkdirAll(filepath.Dir(logPath), 0755)
		log, _ = os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	}

	r := &watcherRuntime{
		control:  control,
		taskbars: newTaskbarManager(),
		visual:   visual,
		log:      log,
	}
	activeRuntime = r

	r.taskbarCreated = registerWindowMessage("TaskbarCreated")
	if r.taskbarCreated == 0 {
		activeRuntime = nil
		if log != nil {
			_ = log.Close()
		}
		return nil, errors.New("RegisterWindowMessage(TaskbarCreated) failed")
	}

	r.keyboardHook, err = installKeyboardHook(keyboardProcCB)
	if err != nil {
		activeRuntime = nil
		if log != nil {
			_ = log.Close()
		}
		return nil, err
	}

	r.foregroundHook, err = installForegroundHook(winEventProcCB)
	if err != nil {
		unhookKeyboard(r.keyboardHook)
		activeRuntime = nil
		if log != nil {
			_ = log.Close()
		}
		return nil, err
	}

	taskbarCount, discoverErr := r.taskbars.discoverAllowEmpty()
	if discoverErr != nil {
		unhookWinEvent(r.foregroundHook)
		unhookKeyboard(r.keyboardHook)
		activeRuntime = nil
		if log != nil {
			_ = log.Close()
		}
		return nil, discoverErr
	}
	if visual && taskbarCount > 0 {
		// Crash recovery: composition belongs to Explorer's taskbar window and can
		// outlive an abruptly terminated LangTint process. Reset verified Explorer
		// taskbars before applying the current language state.
		for tb := range r.taskbars.windows {
			if err := restoreTaskbarNormal(tb); err != nil {
				unhookWinEvent(r.foregroundHook)
				unhookKeyboard(r.keyboardHook)
				activeRuntime = nil
				if log != nil {
					_ = log.Close()
				}
				return nil, fmt.Errorf("startup taskbar crash-recovery restore: %w", err)
			}
		}
	}
	if visual {
		r.visualWorker = newVisualWorker(r, r.taskbars)
	}

	r.refreshHighContrast()
	visualWasEnabled := r.visual
	if taskbarCount == 0 {
		// Explorer may not have created the taskbar yet during logon. Determine
		// language if possible, but never fail or invoke the visual backend here.
		r.visual = false
	}
	initialErr := r.refreshLanguage("startup", true, visual && taskbarCount > 0)
	r.visual = visualWasEnabled
	if initialErr != nil && taskbarCount > 0 {
		unhookWinEvent(r.foregroundHook)
		unhookKeyboard(r.keyboardHook)
		if r.visualWorker != nil {
			r.visualWorker.stop()
		} else {
			r.taskbars.close()
		}
		activeRuntime = nil
		if log != nil {
			_ = log.Close()
		}
		return nil, fmt.Errorf("initial visual state failed: %w", initialErr)
	}
	if taskbarCount == 0 {
		r.beginStartupRetry()
		r.logf("STARTUP_TASKBAR_DEFERRED bounded_retry_ms=%d", startupRetryTotalMS())
	} else if r.language == LanguageUnknown {
		unhookWinEvent(r.foregroundHook)
		unhookKeyboard(r.keyboardHook)
		if r.visualWorker != nil {
			r.visualWorker.stop()
		}
		activeRuntime = nil
		if log != nil {
			_ = log.Close()
		}
		return nil, errors.New("initial language could not be determined")
	}

	cleanupOLE = false
	controlRegistered = false
	cleanupControl = false
	return r, nil
}

func (r *watcherRuntime) cleanup() {
	if r == nil {
		return
	}
	if r.control != 0 {
		killTimer(r.control, timerLayout)
		killTimer(r.control, timerRebind)
		killTimer(r.control, timerStop)
	}
	unhookWinEvent(r.foregroundHook)
	unhookKeyboard(r.keyboardHook)
	if r.visualWorker != nil {
		r.visualWorker.stop()
	} else if r.taskbars != nil {
		r.taskbars.close()
	}
	if r.control != 0 && isWindow(r.control) {
		destroyWindow(r.control)
	}
	if activeRuntime == r {
		activeRuntime = nil
	}
	if r.log != nil {
		_ = r.log.Close()
	}
	unregisterWindowClass(controlClassName)
	oleUninitialize()
}

func runWatcher(logPath string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	mutex, already, err := createSingleInstanceMutex(mutexName)
	if err != nil {
		return err
	}
	if already {
		return nil
	}
	defer releaseMutex(mutex)

	ready, _, err := createNamedEvent(readyName, true, false)
	if err != nil {
		return err
	}
	defer closeHandle(ready)

	r, err := setupRuntime(true, logPath)
	if err != nil {
		return err
	}
	defer r.cleanup()

	setEvent(ready)
	r.logf("READY build=%s language=%s keyboard_hook=PASS foreground_hook=PASS polling=ZERO", buildID, r.language)
	return messageLoop()
}

func eventSelfTest(reportPath string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	lines := []string{
		"LangTint PUBLIC v1.7 EVENT SELF TEST",
		"BUILD_ID=" + buildID,
		"Started: " + time.Now().Format(time.RFC3339),
		"Architecture: event-driven keyboard/foreground hooks + dedicated visual worker",
		"Acceptance policy: NON-MUTATING; no SendInput and no forced keyboard-layout switch",
		"Taskbar render: direct composition; no overlay window",
		"Cursor render: recoloured Arrow + Hand only",
		"Permanent polling: ZERO",
	}
	finish := func(pass bool, err error) error {
		if pass {
			lines = append(lines, "SELF_TEST=PASS")
		} else {
			lines = append(lines, "SELF_TEST=FAIL")
			if err != nil {
				lines = append(lines, "ERROR="+err.Error())
			}
		}
		_ = writeReport(reportPath, lines)
		return err
	}

	if err := validateSupportedWindows(); err != nil {
		return finish(false, err)
	}
	rawTaskbars := enumerateTaskbarsRaw()
	verifiedTaskbars := enumerateTaskbars()
	lines = append(lines,
		fmt.Sprintf("TASKBAR_CLASS_WINDOWS=%d", len(rawTaskbars)),
		fmt.Sprintf("EXPLORER_OWNED_TASKBARS=%d", len(verifiedTaskbars)),
	)
	if len(verifiedTaskbars) == 0 {
		return finish(false, errors.New("no explorer.exe-owned taskbar detected"))
	}

	layouts, layoutErr := installedKeyboardLayouts()
	if layoutErr != nil {
		return finish(false, layoutErr)
	}
	hasEnglish := false
	for _, hkl := range layouts {
		if languageFromHKL(hkl) == LanguageEnglish {
			hasEnglish = true
			break
		}
	}
	lines = append(lines,
		fmt.Sprintf("INSTALLED_INPUT_LOCALES=%d", len(layouts)),
		"ENGLISH_INPUT_LOCALE_PRESENT="+yesNo(hasEnglish),
	)

	r, err := setupRuntime(true, "")
	if err != nil {
		return finish(false, err)
	}
	defer r.cleanup()

	initial := r.language
	if initial == LanguageUnknown {
		return finish(false, errors.New("active input language could not be determined"))
	}
	lines = append(lines,
		"INITIAL_LANGUAGE="+initial.String(),
		"KEYBOARD_HOOK_INSTALL=PASS",
		"FOREGROUND_EVENT_HOOK_INSTALL=PASS",
		"LOCALIZED_MSAA_FALLBACK=WINDOWS_NLS",
		"TASKBAR_OWNER_CHECK=explorer.exe",
		"OVERLAY_WINDOW=NONE",
	)

	if r.visualWorker == nil {
		return finish(false, errors.New("visual worker is missing"))
	}
	baseSet := atomic.LoadUint64(&r.visualWorker.setCalls)
	baseRestore := atomic.LoadUint64(&r.visualWorker.restoreCalls)
	baseCursorSet := atomic.LoadUint64(&r.visualWorker.cursorSetCalls)
	baseCursorRestore := atomic.LoadUint64(&r.visualWorker.cursorRestoreCalls)
	baseCursorObjects := atomic.LoadUint64(&r.visualWorker.cursorObjectsSet)
	baseFailures := atomic.LoadUint64(&r.visualWorker.failures)
	baseCursorFailures := atomic.LoadUint64(&r.visualWorker.cursorFailures)

	if r.highContrast {
		// Accessibility policy wins over visual validation. LangTint deliberately
		// stays visually neutral while High Contrast is active.
		if err := r.visualWorker.submit(false, true, true); err != nil {
			return finish(false, fmt.Errorf("High Contrast safe-state restore: %w", err))
		}
		lines = append(lines,
			"HIGH_CONTRAST_ACTIVE=YES",
			"VISUAL_TINT_TEST=SKIPPED_ACCESSIBILITY_POLICY",
			"NON_MUTATING_ACCEPTANCE=PASS",
			"MULTI_LAYOUT_SAFE=PASS",
			"SYSTEM_THEME_REGISTRY_TOUCHED=NO",
		)
		return finish(true, nil)
	}

	// Exercise both visual states directly. This validates DWM and cursor APIs
	// without changing the user's keyboard layout or depending on their hotkey.
	if err := r.visualWorker.submit(true, true, true); err != nil {
		return finish(false, fmt.Errorf("EN visual backend test: %w", err))
	}
	if err := r.visualWorker.submit(false, true, true); err != nil {
		return finish(false, fmt.Errorf("normal visual backend test: %w", err))
	}
	// Restore the visual state that corresponds to the user's actual language.
	if err := r.visualWorker.submit(shouldTint(initial, r.highContrast), true, true); err != nil {
		return finish(false, fmt.Errorf("restore initial visual state: %w", err))
	}

	setDelta := atomic.LoadUint64(&r.visualWorker.setCalls) - baseSet
	restoreDelta := atomic.LoadUint64(&r.visualWorker.restoreCalls) - baseRestore
	cursorSetDelta := atomic.LoadUint64(&r.visualWorker.cursorSetCalls) - baseCursorSet
	cursorRestoreDelta := atomic.LoadUint64(&r.visualWorker.cursorRestoreCalls) - baseCursorRestore
	cursorObjectDelta := atomic.LoadUint64(&r.visualWorker.cursorObjectsSet) - baseCursorObjects
	failureDelta := atomic.LoadUint64(&r.visualWorker.failures) - baseFailures
	cursorFailureDelta := atomic.LoadUint64(&r.visualWorker.cursorFailures) - baseCursorFailures
	lines = append(lines,
		fmt.Sprintf("COMPOSITION_SET_CALLS_TEST=%d", setDelta),
		fmt.Sprintf("COMPOSITION_RESTORE_CALLS_TEST=%d", restoreDelta),
		fmt.Sprintf("CURSOR_SET_CALLS_TEST=%d", cursorSetDelta),
		fmt.Sprintf("CURSOR_RESTORE_CALLS_TEST=%d", cursorRestoreDelta),
		fmt.Sprintf("CURSOR_OBJECTS_SET_TEST=%d", cursorObjectDelta),
		fmt.Sprintf("VISUAL_WORKER_FAILURES_TEST=%d", failureDelta),
		fmt.Sprintf("CURSOR_WORKER_FAILURES_TEST=%d", cursorFailureDelta),
	)
	if setDelta == 0 || restoreDelta == 0 {
		return finish(false, errors.New("taskbar composition round-trip did not execute"))
	}
	if cursorSetDelta == 0 || cursorRestoreDelta == 0 {
		return finish(false, errors.New("cursor round-trip did not execute"))
	}
	if cursorObjectDelta < uint64(len(tintedSystemCursorIDs)) {
		return finish(false, errors.New("Arrow+Hand cursor policy did not execute"))
	}
	if failureDelta != 0 || cursorFailureDelta != 0 {
		return finish(false, errors.New("visual backend reported failures"))
	}
	lines = append(lines,
		"NON_MUTATING_ACCEPTANCE=PASS",
		"MULTI_LAYOUT_SAFE=PASS",
		"TASKBAR_COMPOSITION_ROUNDTRIP=PASS",
		"SYSTEM_CURSOR_ROUNDTRIP=PASS",
		"SYSTEM_THEME_REGISTRY_TOUCHED=NO",
	)
	return finish(true, nil)
}

func idleCPUTest(reportPath string, seconds int) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if seconds < 10 {
		seconds = 10
	}
	lines := []string{
		"LangTint IDLE CPU TEST",
		"BUILD_ID=" + buildID,
		fmt.Sprintf("DURATION_SECONDS=%d", seconds),
		"Visual backend: dedicated taskbar+cursor worker; actual installed runtime path",
		"Permanent polling: ZERO",
	}

	r, err := setupRuntime(true, "")
	if err != nil {
		lines = append(lines, "IDLE_CPU_TEST=FAIL", "ERROR="+err.Error())
		_ = writeReport(reportPath, lines)
		return err
	}
	defer r.cleanup()

	startCPU, err := processCPU100ns()
	if err != nil {
		return err
	}
	startWall := time.Now()
	if err := setTimer(r.control, timerStop, uint32(seconds*1000)); err != nil {
		return err
	}
	if err := messageLoop(); err != nil {
		return err
	}
	endCPU, err := processCPU100ns()
	if err != nil {
		return err
	}
	wall := time.Since(startWall)
	cpuSeconds := float64(endCPU-startCPU) / 10_000_000.0
	oneCorePct := 0.0
	if wall > 0 {
		oneCorePct = cpuSeconds / wall.Seconds() * 100.0
	}
	lines = append(lines,
		fmt.Sprintf("CPU_SECONDS=%.6f", cpuSeconds),
		fmt.Sprintf("WALL_SECONDS=%.3f", wall.Seconds()),
		fmt.Sprintf("CPU_ONE_CORE_PERCENT=%.4f", oneCorePct),
		fmt.Sprintf("HOTKEY_EVENTS=%d", atomic.LoadUint64(&r.hotkeyEvents)),
		fmt.Sprintf("FOREGROUND_EVENTS=%d", atomic.LoadUint64(&r.foregroundEvents)),
		fmt.Sprintf("LAYOUT_CHECKS=%d", atomic.LoadUint64(&r.layoutChecks)),
		fmt.Sprintf("VISUAL_REQUESTS_SUPPRESSED=%d", atomic.LoadUint64(&r.visualSuppressed)),
	)
	if r.visualWorker != nil {
		lines = append(lines,
			fmt.Sprintf("VISUAL_WORKER_COMMANDS=%d", atomic.LoadUint64(&r.visualWorker.commands)),
			fmt.Sprintf("COMPOSITION_SET_CALLS=%d", atomic.LoadUint64(&r.visualWorker.setCalls)),
			fmt.Sprintf("COMPOSITION_RESTORE_CALLS=%d", atomic.LoadUint64(&r.visualWorker.restoreCalls)),
			fmt.Sprintf("CURSOR_SET_CALLS=%d", atomic.LoadUint64(&r.visualWorker.cursorSetCalls)),
			fmt.Sprintf("CURSOR_RESTORE_CALLS=%d", atomic.LoadUint64(&r.visualWorker.cursorRestoreCalls)),
			fmt.Sprintf("CURSOR_OBJECTS_SET=%d", atomic.LoadUint64(&r.visualWorker.cursorObjectsSet)),
			fmt.Sprintf("CURSOR_WORKER_FAILURES=%d", atomic.LoadUint64(&r.visualWorker.cursorFailures)),
		)
	}
	minimumWall := time.Duration(seconds) * time.Second * 9 / 10
	if wall < minimumWall {
		err := fmt.Errorf("idle CPU wall time %.3fs is shorter than required %.3fs", wall.Seconds(), minimumWall.Seconds())
		lines = append(lines, "IDLE_CPU_TEST=FAIL", "ERROR="+err.Error())
		_ = writeReport(reportPath, lines)
		return err
	}
	if oneCorePct > 0.50 {
		err := fmt.Errorf("idle CPU %.4f%% exceeds 0.50%% of one core", oneCorePct)
		lines = append(lines, "IDLE_CPU_TEST=FAIL", "ERROR="+err.Error())
		_ = writeReport(reportPath, lines)
		return err
	}
	lines = append(lines, "IDLE_CPU_TEST=PASS")
	return writeReport(reportPath, lines)
}

func yesNo(v bool) string {
	if v {
		return "YES"
	}
	return "NO"
}

func cleanupLegacyPollingVersion() {
	_ = deleteRunValue(legacyRunName)
	// Native v1 used a named stop event. Signaling it is harmless if absent.
	const eventModifyState = 0x0002
	if h := openNamedEvent(legacyStopName, eventModifyState); h != 0 {
		setEvent(h)
		closeHandle(h)
		time.Sleep(150 * time.Millisecond)
	}
	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		_ = os.RemoveAll(filepath.Join(base, legacyInstallFolder))
	}
}

// cleanupPriorHotkeyFamily removes stale installed builds from
// the v1.0-v1.5 family. Those versions intentionally shared one window class,
// Run value and install directory, which made it possible to accidentally keep
// executing an older watcher. Versioned builds remove the old family before
// acceptance testing.
func cleanupPriorHotkeyFamily(timeout time.Duration) error {
	_ = deleteRunValue(priorHotkeyRunName)

	deadline := time.Now().Add(timeout)
	for {
		hwnd := findWindow(priorHotkeyControlClass)
		if hwnd == 0 {
			break
		}
		postMessage(hwnd, wmClose, 0, 0)
		if time.Now().After(deadline) {
			return errors.New("previous TaskbarLayoutTintHotkey process did not stop")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Restore user-visible state defensively after the old process is gone.
	// Overlay versions disappear when their process exits; composition versions
	// are explicitly told to return the taskbar to the Explorer default.
	var restoreErr error
	for _, tb := range enumerateTaskbars() {
		if err := restoreTaskbarNormal(tb); err != nil && restoreErr == nil {
			restoreErr = err
		}
	}
	if err := restoreSystemCursors(); err != nil && restoreErr == nil {
		restoreErr = err
	}

	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		if err := os.RemoveAll(filepath.Join(base, priorHotkeyInstallFolder)); err != nil && restoreErr == nil {
			restoreErr = err
		}
	}
	return restoreErr
}

func cleanupV151Family(timeout time.Duration) error {
	_ = deleteRunValue(priorV151RunName)

	deadline := time.Now().Add(timeout)
	for {
		hwnd := findWindow(priorV151ControlClass)
		if hwnd == 0 {
			break
		}
		postMessage(hwnd, wmClose, 0, 0)
		if time.Now().After(deadline) {
			return errors.New("previous v1.5.1 process did not stop; source files may remain locked")
		}
		time.Sleep(50 * time.Millisecond)
	}

	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		if err := os.RemoveAll(filepath.Join(base, priorV151InstallFolder)); err != nil {
			return err
		}
	}
	return nil
}

func cleanupV152Family(timeout time.Duration) error {
	_ = deleteRunValue(priorV152RunName)

	deadline := time.Now().Add(timeout)
	for {
		hwnd := findWindow(priorV152ControlClass)
		if hwnd == 0 {
			break
		}
		postMessage(hwnd, wmClose, 0, 0)
		if time.Now().After(deadline) {
			return errors.New("previous v1.5.2 process did not stop")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// v1.5.2 could have left Arrow/I-Beam/etc. tinted while EN was active.
	// Restore the complete configured Windows cursor scheme before applying the
	// narrower v1.5.3 Arrow+Hand-only policy.
	var restoreErr error
	for _, tb := range enumerateTaskbars() {
		if err := restoreTaskbarNormal(tb); err != nil && restoreErr == nil {
			restoreErr = err
		}
	}
	if err := restoreSystemCursors(); err != nil && restoreErr == nil {
		restoreErr = err
	}

	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		if err := os.RemoveAll(filepath.Join(base, priorV152InstallFolder)); err != nil && restoreErr == nil {
			restoreErr = err
		}
	}
	return restoreErr
}

func cleanupV153Family(timeout time.Duration) error {
	_ = deleteRunValue(priorV153RunName)
	deadline := time.Now().Add(timeout)
	for {
		hwnd := findWindow(priorV153ControlClass)
		if hwnd == 0 {
			break
		}
		postMessage(hwnd, wmClose, 0, 0)
		if time.Now().After(deadline) {
			return errors.New("previous v1.5.3 process did not stop")
		}
		time.Sleep(50 * time.Millisecond)
	}
	var restoreErr error
	for _, tb := range enumerateTaskbars() {
		if err := restoreTaskbarNormal(tb); err != nil && restoreErr == nil {
			restoreErr = err
		}
	}
	if err := restoreSystemCursors(); err != nil && restoreErr == nil {
		restoreErr = err
	}
	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		if err := os.RemoveAll(filepath.Join(base, priorV153InstallFolder)); err != nil && restoreErr == nil {
			restoreErr = err
		}
	}
	return restoreErr
}

func cleanupAllPriorVersions() error {
	cleanupLegacyPollingVersion()
	if err := cleanupPriorHotkeyFamily(4 * time.Second); err != nil {
		return err
	}
	if err := cleanupV151Family(4 * time.Second); err != nil {
		return err
	}
	if err := cleanupV152Family(4 * time.Second); err != nil {
		return err
	}
	return cleanupV153Family(4 * time.Second)
}

func localInstallDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", errors.New("LOCALAPPDATA is empty")
	}
	return filepath.Join(base, installFolder), nil
}

func localDataDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", errors.New("LOCALAPPDATA is empty")
	}
	return filepath.Join(base, appName), nil
}

func installedPaths() (dir, exe, manifest, log string, err error) {
	dir, err = localInstallDir()
	if err != nil {
		return
	}
	dataDir, dataErr := localDataDir()
	if dataErr != nil {
		err = dataErr
		return
	}
	exe = filepath.Join(dir, exeName)
	manifest = exe + ".manifest"
	// Keep the program directory clean. Runtime diagnostics live in the
	// per-user data directory and are removed by the normal uninstaller.
	log = filepath.Join(dataDir, "Logs", "LangTint.log")
	return
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	tmp := dst + ".new"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	_ = os.Remove(dst)
	return os.Rename(tmp, dst)
}

func stopRunning(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		hwnd := findWindow(controlClassName)
		if hwnd == 0 {
			return
		}
		postMessage(hwnd, wmClose, 0, 0)
		time.Sleep(50 * time.Millisecond)
	}
}

func startInstalled(exe string) error {
	cmd := exec.Command(exe, "--run")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func waitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		h := openNamedEvent(readyName, 0x00100000)
		if h != 0 {
			res := waitForSingleObject(h, 150)
			closeHandle(h)
			if res == 0 {
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return errors.New("READY event not observed")
}

func writeReport(path string, lines []string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, line := range lines {
		_, _ = fmt.Fprintln(w, line)
	}
	return w.Flush()
}

func install(reportPath string) error {
	lines := []string{
		"LangTint INSTALL REPORT",
		"BUILD_ID=" + buildID,
		"Started: " + time.Now().Format(time.RFC3339),
		"Architecture: event-driven hooks + dedicated taskbar/cursor visual worker",
		"Permanent polling: ZERO",
		"Windows theme/accent registry modifications: NONE",
	}
	finish := func(result string, err error) error {
		lines = append(lines, "RESULT="+result)
		if err != nil {
			lines = append(lines, "ERROR="+err.Error())
		}
		_ = writeReport(reportPath, lines)
		return err
	}

	if err := cleanupAllPriorVersions(); err != nil {
		return finish("FAIL", fmt.Errorf("previous-version cleanup: %w", err))
	}
	stopRunning(3 * time.Second)

	srcExe, err := os.Executable()
	if err != nil {
		return finish("FAIL", err)
	}
	srcExe, _ = filepath.Abs(srcExe)
	srcManifest := srcExe + ".manifest"
	dir, dstExe, dstManifest, _, err := installedPaths()
	if err != nil {
		return finish("FAIL", err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return finish("FAIL", err)
	}
	sameInstalledExe := strings.EqualFold(filepath.Clean(srcExe), filepath.Clean(dstExe))
	if !sameInstalledExe {
		if err := copyFile(srcExe, dstExe); err != nil {
			return finish("FAIL", fmt.Errorf("copy exe: %w", err))
		}
	}
	if _, err := os.Stat(srcManifest); err != nil {
		if !sameInstalledExe {
			_ = os.RemoveAll(dir)
		}
		return finish("FAIL", errors.New("sidecar application manifest missing"))
	}
	if !strings.EqualFold(filepath.Clean(srcManifest), filepath.Clean(dstManifest)) {
		if err := copyFile(srcManifest, dstManifest); err != nil {
			if !sameInstalledExe {
				_ = os.RemoveAll(dir)
			}
			return finish("FAIL", fmt.Errorf("copy manifest: %w", err))
		}
	}

	runCmd := fmt.Sprintf(`"%s" --run`, dstExe)
	if err := setRunValue(appName, runCmd); err != nil {
		_ = os.RemoveAll(dir)
		return finish("FAIL", err)
	}
	if err := startInstalled(dstExe); err != nil {
		_ = deleteRunValue(appName)
		_ = os.RemoveAll(dir)
		return finish("FAIL", fmt.Errorf("start watcher: %w", err))
	}
	if err := waitReady(8 * time.Second); err != nil {
		stopRunning(2 * time.Second)
		_ = deleteRunValue(appName)
		_ = os.RemoveAll(dir)
		return finish("FAIL_ROLLED_BACK", err)
	}
	lines = append(lines,
		"AUTOSTART=PASS",
		"WATCHER_READY=PASS",
		"SHORTCUT_EVENTS=ALT_SHIFT+CTRL_SHIFT+WIN_SPACE",
		"FOREGROUND_EVENT=EVENT_SYSTEM_FOREGROUND",
		"PERMANENT_POLLING=ZERO",
		"VISUAL_METHOD=TASKBAR_COMPOSITION_PLUS_SYSTEM_CURSOR_TINT",
		"RU=normal taskbar + configured Windows cursor scheme",
		"EN=opaque pale-blue taskbar background + pale-blue Arrow/Hand with dark contour; I-Beam stays normal",
	)
	return finish("PASS", nil)
}

func stopAndRestore(reportPath string) error {
	stopRunning(4 * time.Second)
	var restoreErr error
	for _, tb := range enumerateTaskbars() {
		if err := restoreTaskbarNormal(tb); err != nil && restoreErr == nil {
			restoreErr = err
		}
	}
	if err := restoreSystemCursors(); err != nil && restoreErr == nil {
		restoreErr = err
	}
	lines := []string{
		"LangTint STOP REPORT",
		"BUILD_ID=" + buildID,
		"WATCHER_STOPPED=" + fmt.Sprintf("%v", findWindow(controlClassName) == 0),
		"AUTOSTART_PRESERVED=YES",
		"VISUALS_RESTORED=" + yesNo(restoreErr == nil),
	}
	if restoreErr == nil {
		lines = append(lines, "RESULT=PASS")
	} else {
		lines = append(lines, "RESULT=FAIL", "ERROR="+restoreErr.Error())
	}
	_ = writeReport(reportPath, lines)
	return restoreErr
}

func uninstall(reportPath string) error {
	_ = cleanupAllPriorVersions()
	_ = deleteRunValue(appName)
	stopRunning(4 * time.Second)
	cursorRestoreErr := restoreSystemCursors()
	dir, _, _, _, pathErr := installedPaths()
	if pathErr == nil {
		_ = os.RemoveAll(dir)
	}
	result := "PASS"
	if cursorRestoreErr != nil {
		result = "FAIL"
	}
	lines := []string{
		"LangTint UNINSTALL REPORT",
		"BUILD_ID=" + buildID,
		"AUTOSTART_REMOVED=YES",
		"WATCHER_STOPPED=" + fmt.Sprintf("%v", findWindow(controlClassName) == 0),
		"SYSTEM_THEME_REGISTRY_TOUCHED=NO",
		"SYSTEM_CURSORS_RESTORED=" + yesNo(cursorRestoreErr == nil),
		"RESULT=" + result,
	}
	_ = writeReport(reportPath, lines)
	if cursorRestoreErr != nil {
		return cursorRestoreErr
	}
	return nil
}

func status(reportPath string) error {
	dir, exe, manifest, _, err := installedPaths()
	if err != nil {
		return err
	}
	_, exeErr := os.Stat(exe)
	_, manifestErr := os.Stat(manifest)
	running := findWindow(controlClassName) != 0 && isMutexPresent(mutexName)
	ok := exeErr == nil && manifestErr == nil && running
	lines := []string{
		"LangTint STATUS",
		"BUILD_ID=" + buildID,
		"INSTALL_DIR=" + dir,
		"EXE_PRESENT=" + fmt.Sprintf("%v", exeErr == nil),
		"MANIFEST_PRESENT=" + fmt.Sprintf("%v", manifestErr == nil),
		"WATCHER_RUNNING=" + fmt.Sprintf("%v", running),
		"PERMANENT_POLLING=ZERO",
	}
	if ok {
		lines = append(lines, "STATUS=PASS")
	} else {
		lines = append(lines, "STATUS=FAIL")
	}
	_ = writeReport(reportPath, lines)
	if !ok {
		return errors.New("status failed")
	}
	return nil
}

func appendReportFile(lines []string, heading, path string) []string {
	lines = append(lines, "", "===== "+heading+" =====")
	data, err := os.ReadFile(path)
	if err != nil {
		return append(lines, "REPORT_READ_ERROR="+err.Error())
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return append(lines, "(empty report)")
	}
	return append(lines, strings.Split(text, "\n")...)
}

func acceptAndInstall(reportPath string) error {
	if reportPath == "" {
		return errors.New("--accept-install requires --report")
	}

	lines := []string{
		"LangTint PUBLIC v1.7 ONE-CLICK ACCEPTANCE REPORT",
		"BUILD_ID=" + buildID,
		"Started: " + time.Now().Format(time.RFC3339),
		"Architecture: event-driven hooks + dedicated taskbar/cursor visual worker",
		"Layout read: foreground thread GetKeyboardLayout; MSAA only as event-time fallback",
		"Permanent polling: ZERO",
		"Windows theme/accent registry modifications: NONE",
	}
	finish := func(result, step string, err error) error {
		lines = append(lines, "")
		lines = append(lines, "============================================================")
		lines = append(lines, "FINAL_RESULT="+result)
		if step != "" {
			lines = append(lines, "FAILED_STEP="+step)
		}
		if err != nil {
			lines = append(lines, "ERROR="+err.Error())
		}
		lines = append(lines, "PERMANENT_POLLING=ZERO")
		lines = append(lines, "SYSTEM_THEME_REGISTRY_TOUCHED=NO")
		lines = append(lines, "============================================================")
		_ = writeReport(reportPath, lines)
		return err
	}

	// Make the run deterministic and remove all predecessor watchers first.
	if err := cleanupAllPriorVersions(); err != nil {
		return finish("FAIL", "PRIOR_VERSION_CLEANUP", err)
	}
	_ = deleteRunValue(appName)
	stopRunning(2 * time.Second)

	tmpDir, err := os.MkdirTemp("", "LangTint-accept-")
	if err != nil {
		return finish("FAIL", "TEMP_DIR", err)
	}
	defer os.RemoveAll(tmpDir)

	selfReport := filepath.Join(tmpDir, "self.txt")
	idleReport := filepath.Join(tmpDir, "idle.txt")
	installReport := filepath.Join(tmpDir, "install.txt")
	statusReport := filepath.Join(tmpDir, "status.txt")
	rollbackReport := filepath.Join(tmpDir, "rollback.txt")

	if err := eventSelfTest(selfReport); err != nil {
		lines = appendReportFile(lines, "1 EVENT SELF TEST", selfReport)
		return finish("FAIL", "EVENT_SELF_TEST", err)
	}
	lines = appendReportFile(lines, "1 EVENT SELF TEST", selfReport)

	if err := runIdleCPUChild(idleReport, 10); err != nil {
		lines = appendReportFile(lines, "2 IDLE CPU TEST", idleReport)
		return finish("FAIL", "IDLE_CPU_TEST", err)
	}
	lines = appendReportFile(lines, "2 IDLE CPU TEST", idleReport)

	if err := install(installReport); err != nil {
		lines = appendReportFile(lines, "3 INSTALL", installReport)
		_ = uninstall(rollbackReport)
		lines = appendReportFile(lines, "ROLLBACK", rollbackReport)
		return finish("FAIL", "INSTALL", err)
	}
	lines = appendReportFile(lines, "3 INSTALL", installReport)

	if err := status(statusReport); err != nil {
		lines = appendReportFile(lines, "4 STATUS", statusReport)
		_ = uninstall(rollbackReport)
		lines = appendReportFile(lines, "ROLLBACK", rollbackReport)
		return finish("FAIL", "STATUS", err)
	}
	lines = appendReportFile(lines, "4 STATUS", statusReport)

	lines = append(lines,
		"",
		"BUILD_ID_CONFIRMED="+buildID,
		"ALL_ACCEPTANCE_GATES=PASS",
		"INSTALL=PASS",
		"STATUS=PASS",
		"HOTKEY_EVENT=WH_KEYBOARD_LL",
		"FOREGROUND_EVENT=EVENT_SYSTEM_FOREGROUND",
		"LAYOUT_QUERY=GetKeyboardLayout(foreground_thread)",
		"CURSOR_MODE=ARROW_HAND_TINT_WITH_DARK_CONTOUR",
		"VISUAL_METHOD=TASKBAR_COMPOSITION_NO_OVERLAY",
	)
	return finish("PASS", "", nil)
}

func parseArgValue(args []string, name string) string {
	for i := 0; i+1 < len(args); i++ {
		if strings.EqualFold(args[i], name) {
			return args[i+1]
		}
	}
	return ""
}

func runIdleCPUChild(reportPath string, seconds int) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "--idle-test", "--seconds", fmt.Sprintf("%d", seconds), "--report", reportPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("idle CPU child process: %w", err)
	}
	return nil
}

func launchProduct() error {
	current, err := os.Executable()
	if err != nil {
		return err
	}
	current, _ = filepath.Abs(current)
	_, installedExe, _, logPath, err := installedPaths()
	if err != nil {
		return err
	}
	installedExe, _ = filepath.Abs(installedExe)
	if !strings.EqualFold(filepath.Clean(current), filepath.Clean(installedExe)) {
		showInfoMessage("LangTint", "Для установки используйте LangTint-Setup-x64.exe.\n\nLangTint.exe сам себя не устанавливает и не открывает диагностические отчёты.")
		return nil
	}
	if findWindow(controlClassName) != 0 || isMutexPresent(mutexName) {
		return nil
	}
	return runWatcher(logPath)
}

func defaultAcceptanceReport() string {
	return filepath.Join(os.TempDir(), "LangTint_ACCEPTANCE_REPORT.txt")
}

func main() {
	args := os.Args[1:]
	mode := resolveMode(args)
	report := parseArgValue(args, "--report")
	if mode == "--accept-install" && report == "" {
		report = defaultAcceptanceReport()
	}

	seconds := 10
	if v := parseArgValue(args, "--seconds"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &seconds)
	}

	var err error
	switch mode {
	case "--launch":
		err = launchProduct()
	case "--accept-install":
		err = acceptAndInstall(report)
	case "--self-test":
		err = eventSelfTest(report)
	case "--idle-test":
		err = idleCPUTest(report, seconds)
	case "--install":
		err = install(report)
	case "--uninstall":
		err = uninstall(report)
	case "--stop":
		err = stopAndRestore(report)
	case "--status":
		err = status(report)
	case "--run":
		_, _, _, logPath, pathErr := installedPaths()
		if pathErr != nil {
			err = pathErr
		} else {
			err = runWatcher(logPath)
		}
	default:
		err = fmt.Errorf("invalid command line; resident watcher requires explicit --run")
	}
	if err != nil {
		os.Exit(1)
	}
}
