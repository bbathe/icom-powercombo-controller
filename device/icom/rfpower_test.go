package icom

import "testing"

func TestRFPowerToCIV(t *testing.T) {
	tests := []struct {
		percent int
		want    int
	}{
		{0, 0},
		{1, 3},   // 255/100 rounds up
		{30, 77}, // 7650/100 = 76 rem 50 -> 77
		{100, 255},
	}

	for _, tt := range tests {
		got := RFPowerToCIV(tt.percent)
		if got != tt.want {
			t.Fatalf("RFPowerToCIV(%d)=%d, want %d", tt.percent, got, tt.want)
		}
	}
}
