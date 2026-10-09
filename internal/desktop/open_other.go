//go:build !windows

package desktop

import (
	"fmt"
	"os/exec"
	"runtime"
)

// open runs open (macOS) or xdg-open (Linux and others) directly, without a shell.
func open(target string) error {
	name, args := openCommand(runtime.GOOS, target)
	if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("desktop: %s: %w: %s", name, err, out)
	}
	return nil
}
