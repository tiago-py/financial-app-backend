package service

import (
	"strings"
	"testing"
)

func TestValidateNewPassword(t *testing.T) {
	cases := []struct {
		name, current, password string
		valid                   bool
	}{
		{"valid", "old-password", "new-password", true},
		{"blank", "old-password", "        ", false},
		{"short", "old-password", "1234567", false},
		{"same", "old-password", "old-password", false},
		{"bcrypt limit", "old-password", strings.Repeat("a", 72), true},
		{"bcrypt overflow", "old-password", strings.Repeat("a", 73), false},
		{"unicode short", "old-password", "éééé", false},
		{"unicode bytes", "old-password", strings.Repeat("😀", 19), false},
		{"unicode valid", "old-password", strings.Repeat("é", 8), true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateNewPassword(tt.current, tt.password); (got == nil) != tt.valid {
				t.Fatalf("valid = %v, error = %v", tt.valid, got)
			}
		})
	}
}
