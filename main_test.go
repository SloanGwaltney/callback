package main

import (
	"testing"
	"time"

	"github.com/wailsapp/wails/v2/pkg/options"
)

func TestStartHidden(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"unset", "", false},
		{"set", "1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string {
				if key == StartHiddenEnvVar {
					return tt.value
				}
				return ""
			}
			if got := startHidden(getenv); got != tt.want {
				t.Errorf("startHidden() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWindowStartState(t *testing.T) {
	tests := []struct {
		name   string
		hidden string
		want   options.WindowStartState
	}{
		{"opens maximized", "", options.Maximised},
		{"stays normal when hidden", "1", options.Normal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string {
				if key == StartHiddenEnvVar {
					return tt.hidden
				}
				return ""
			}
			if got := windowStartState(getenv); got != tt.want {
				t.Errorf("windowStartState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClock(t *testing.T) {
	actual := time.Date(2026, time.October, 12, 15, 4, 5, 0, time.Local)
	now := func() time.Time { return actual }
	env := func(value string) func(string) string {
		return func(key string) string {
			if key == TodayEnvVar {
				return value
			}
			return ""
		}
	}

	t.Run("unset uses the real time", func(t *testing.T) {
		got, err := clock(env(""), now)
		if err != nil {
			t.Fatal(err)
		}
		if !got().Equal(actual) {
			t.Errorf("clock() = %v, want %v", got(), actual)
		}
	})

	t.Run("a date replaces the day and keeps the time of day", func(t *testing.T) {
		got, err := clock(env("2026-10-01"), now)
		if err != nil {
			t.Fatal(err)
		}
		want := time.Date(2026, time.October, 1, 15, 4, 5, 0, time.Local)
		if !got().Equal(want) {
			t.Errorf("clock() = %v, want %v", got(), want)
		}
	})

	t.Run("rejects anything but a date", func(t *testing.T) {
		for _, value := range []string{"Oct 1", "2026-10-1", "2026-02-30"} {
			if _, err := clock(env(value), now); err == nil {
				t.Errorf("%q: want an error", value)
			}
		}
	})
}
