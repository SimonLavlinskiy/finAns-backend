package colorutil_test

import (
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/pkg/colorutil"
)

func TestLighten(t *testing.T) {
	tests := []struct {
		name   string
		hex    string
		amount float64
		want   string
	}{
		{
			name:   "black 50% → mid grey",
			hex:    "#000000",
			amount: 0.5,
			want:   "#808080",
		},
		{
			name:   "white stays white",
			hex:    "#FFFFFF",
			amount: 0.5,
			want:   "#FFFFFF",
		},
		{
			name:   "amount 0 → no change",
			hex:    "#112250",
			amount: 0,
			want:   "#112250",
		},
		{
			name:   "amount 1 → white",
			hex:    "#112250",
			amount: 1,
			want:   "#FFFFFF",
		},
		{
			name:   "royal blue 40% lighter",
			hex:    "#112250",
			amount: 0.4,
			want:   "#707A96",
		},
		{
			name:   "lowercase hex input",
			hex:    "#3c5070",
			amount: 0,
			want:   "#3C5070",
		},
		{
			name:   "invalid hex → passthrough",
			hex:    "notahex",
			amount: 0.5,
			want:   "notahex",
		},
		{
			name:   "clamp amount above 1",
			hex:    "#112250",
			amount: 2.0,
			want:   "#FFFFFF",
		},
		{
			name:   "clamp amount below 0",
			hex:    "#112250",
			amount: -1.0,
			want:   "#112250",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := colorutil.Lighten(tt.hex, tt.amount)
			if got != tt.want {
				t.Errorf("Lighten(%q, %v) = %q; want %q", tt.hex, tt.amount, got, tt.want)
			}
		})
	}
}
