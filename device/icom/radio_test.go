package icom

import "testing"

func TestParseOperatingFrequency(t *testing.T) {
	// FE FE E0 94 00 [freq BCD LSB-first] FD — 14.250000 MHz as example shape
	msg := []byte{0xFE, 0xFE, 0xE0, 0x94, 0x00, 0x00, 0x00, 0x25, 0x14, 0x00, 0xFD}
	freq, ok := parseOperatingFrequency(msg)
	if !ok {
		t.Fatal("expected frequency message")
	}
	if freq != 14250000 {
		t.Fatalf("freq=%d, want 14250000", freq)
	}

	if _, ok := parseOperatingFrequency([]byte{0xFE, 0xFE}); ok {
		t.Fatal("short message should not parse")
	}
}
