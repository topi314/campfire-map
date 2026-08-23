package server

import "testing"

func TestOriginAllowed(t *testing.T) {
	tests := []struct {
		allowed string
		origin  string
		want    bool
	}{
		{"http://localhost:3000", "http://localhost:3000", true},
		{"http://localhost:3000", "http://127.0.0.1:3000", true},
		{"http://localhost:3000", "http://localhost:3001", true},
		{"http://localhost:3000", "http://127.0.0.1:3001", true},
		{"*", "https://example.com", true},
		{"https://export.example.com", "https://export.example.com", true},
		{"https://export.example.com", "https://evil.example", false},
		{"http://localhost:3000", "", false},
	}
	for _, tt := range tests {
		if got := originAllowed(tt.allowed, tt.origin); got != tt.want {
			t.Errorf("originAllowed(%q, %q)=%v want %v", tt.allowed, tt.origin, got, tt.want)
		}
	}
}
