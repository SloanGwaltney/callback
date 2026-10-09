package desktop

import (
	"slices"
	"strings"
	"testing"
)

// The OS calls themselves can't be unit tested; they're checked by running the app. See
// docs/adr/0017-open-documents-and-links-with-a-hand-written-os-opener.md.

func TestOpenCommand(t *testing.T) {
	tests := []struct {
		goos, name string
	}{
		{"darwin", "open"},
		{"linux", "xdg-open"},
		{"freebsd", "xdg-open"},
	}
	for _, tt := range tests {
		target := "/home/sam/.local/share/callback/documents/a b.pdf"
		name, args := openCommand(tt.goos, target)
		// The target is one argument, never split or passed through a shell.
		if name != tt.name || !slices.Equal(args, []string{target}) {
			t.Errorf("%s: %s %q, want %s [%q]", tt.goos, name, args, tt.name, target)
		}
	}
}

func TestShellExecuteError(t *testing.T) {
	tests := map[uintptr]string{
		2:  "file not found",
		31: "no app is associated with this file type",
		99: "error code 99",
	}
	for code, want := range tests {
		if err := shellExecuteError(code); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("shellExecuteError(%d) = %v, want it to mention %q", code, err, want)
		}
	}
}

func TestOpenRefusesAnEmptyTarget(t *testing.T) {
	if err := (Opener{}).Open(""); err == nil {
		t.Error("opening nothing succeeded")
	}
}
