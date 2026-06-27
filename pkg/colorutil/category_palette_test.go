package colorutil_test

import (
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/pkg/colorutil"
)

func TestIsValidCategoryColor(t *testing.T) {
	tests := []struct {
		name string
		hex  string
		want bool
	}{
		{"exact match", "#112250", true},
		{"case insensitive match", "#e0c68f", true},
		{"not in palette", "#123456", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := colorutil.IsValidCategoryColor(tt.hex)
			if got != tt.want {
				t.Errorf("IsValidCategoryColor(%q) = %v; want %v", tt.hex, got, tt.want)
			}
		})
	}
}
