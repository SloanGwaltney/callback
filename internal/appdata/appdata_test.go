package appdata

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	home := func() (string, error) { return "/home/sam", nil }

	tests := []struct {
		name string
		goos string
		env  map[string]string
		want string
	}{
		{
			name: "override wins on any OS",
			goos: "windows",
			env:  map[string]string{EnvVar: "/tmp/callback-dev", "LOCALAPPDATA": `C:\Users\sam\AppData\Local`},
			want: "/tmp/callback-dev",
		},
		{
			name: "windows uses LOCALAPPDATA",
			goos: "windows",
			env:  map[string]string{"LOCALAPPDATA": `C:\Users\sam\AppData\Local`, "APPDATA": `C:\Users\sam\AppData\Roaming`},
			want: filepath.Join(`C:\Users\sam\AppData\Local`, "callback"),
		},
		{
			name: "macOS uses Application Support",
			goos: "darwin",
			want: filepath.Join("/home/sam", "Library", "Application Support", "callback"),
		},
		{
			name: "linux uses XDG_DATA_HOME when set",
			goos: "linux",
			env:  map[string]string{"XDG_DATA_HOME": "/data/sam"},
			want: filepath.Join("/data/sam", "callback"),
		},
		{
			name: "linux falls back to ~/.local/share",
			goos: "linux",
			want: filepath.Join("/home/sam", ".local", "share", "callback"),
		},
		{
			name: "linux ignores a relative XDG_DATA_HOME",
			goos: "linux",
			env:  map[string]string{"XDG_DATA_HOME": "relative/data"},
			want: filepath.Join("/home/sam", ".local", "share", "callback"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dir(tt.goos, func(k string) string { return tt.env[k] }, home)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDirErrors(t *testing.T) {
	noEnv := func(string) string { return "" }
	homeErr := errors.New("no home")
	noHome := func() (string, error) { return "", homeErr }

	if _, err := dir("windows", noEnv, noHome); err == nil {
		t.Error("windows without LOCALAPPDATA: expected an error")
	}
	if _, err := dir("darwin", noEnv, noHome); !errors.Is(err, homeErr) {
		t.Errorf("darwin without a home dir: got %v, want %v", err, homeErr)
	}
	if _, err := dir("linux", noEnv, noHome); !errors.Is(err, homeErr) {
		t.Errorf("linux without a home dir: got %v, want %v", err, homeErr)
	}
}
