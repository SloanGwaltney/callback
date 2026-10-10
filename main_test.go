package main

import "testing"

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
