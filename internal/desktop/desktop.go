// Package desktop opens files and links in the operating system's default handler. See
// docs/adr/0017-open-documents-and-links-with-a-hand-written-os-opener.md.
package desktop

import "fmt"

// Opener opens a file path or an http(s) URL with the OS default handler: the default app for a
// file's type, or the web browser for a link. Callers check that links are web addresses first.
type Opener struct{}

// Open opens target and reports an error if the OS couldn't.
func (Opener) Open(target string) error {
	if target == "" {
		return fmt.Errorf("desktop: nothing to open")
	}
	return open(target)
}

// openCommand returns the program and arguments that open target on a non-Windows OS.
func openCommand(goos, target string) (string, []string) {
	if goos == "darwin" {
		return "open", []string{target}
	}
	return "xdg-open", []string{target}
}

// shellExecuteError turns a ShellExecuteW return value of 32 or less into an error. Values above
// 32 mean success. See
// https://learn.microsoft.com/windows/win32/api/shellapi/nf-shellapi-shellexecutew#return-value.
func shellExecuteError(code uintptr) error {
	var reason string
	switch code {
	case 0:
		reason = "out of memory or resources"
	case 2:
		reason = "file not found"
	case 3:
		reason = "path not found"
	case 5:
		reason = "access denied"
	case 8:
		reason = "not enough memory"
	case 11:
		reason = "invalid executable"
	case 26:
		reason = "sharing violation"
	case 27:
		reason = "incomplete or invalid file association"
	case 28, 29, 30:
		reason = "the app didn't respond"
	case 31:
		reason = "no app is associated with this file type"
	case 32:
		reason = "a required library wasn't found"
	default:
		reason = fmt.Sprintf("error code %d", code)
	}
	return fmt.Errorf("desktop: couldn't open: %s", reason)
}
