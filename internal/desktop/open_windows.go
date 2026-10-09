package desktop

import (
	"syscall"
	"unsafe"
)

var (
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

// swShowNormal is SW_SHOWNORMAL: open the window at its normal size.
const swShowNormal = 1

// open calls ShellExecuteW(NULL, "open", target, NULL, NULL, SW_SHOWNORMAL), which opens a file
// in its default app or a URL in the default browser, the same as double-clicking in Explorer.
func open(target string) error {
	verb, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	// LazyProc.Call is marked go:uintptrescapes, so the pointers converted to uintptr in its
	// arguments stay valid for the whole call.
	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		0,
		0,
		swShowNormal,
	)
	if ret <= 32 {
		return shellExecuteError(ret)
	}
	return nil
}
