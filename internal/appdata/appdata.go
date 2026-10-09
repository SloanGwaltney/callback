// Package appdata finds the directory where callback keeps the user's data: the SQLite database
// and the documents folder. See
// docs/adr/0009-keep-user-data-in-the-os-conventional-app-data-directory.md.
package appdata

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

// EnvVar overrides the data directory when set, so development runs and tests don't touch the
// user's real data.
const EnvVar = "CALLBACK_DATA_DIR"

const appName = "callback"

// Dir returns the data directory for the current OS. It doesn't create the directory.
func Dir() (string, error) {
	return dir(runtime.GOOS, os.Getenv, os.UserHomeDir)
}

func dir(goos string, getenv func(string) string, homeDir func() (string, error)) (string, error) {
	if override := getenv(EnvVar); override != "" {
		return override, nil
	}

	switch goos {
	case "windows":
		local := getenv("LOCALAPPDATA")
		if local == "" {
			return "", errors.New("appdata: %LOCALAPPDATA% is not set")
		}
		return filepath.Join(local, appName), nil

	case "darwin":
		home, err := homeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", appName), nil

	default:
		// The XDG Base Directory spec says to ignore XDG_DATA_HOME unless it's an absolute path.
		// XDG paths are POSIX paths, so check them with path rather than filepath.
		if xdg := getenv("XDG_DATA_HOME"); path.IsAbs(xdg) {
			return filepath.Join(xdg, appName), nil
		}
		home, err := homeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", appName), nil
	}
}
