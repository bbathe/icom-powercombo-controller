package icom

import "testing"

func TestParseOperatingFrequency(t *testing.T) {
	// FE FE E0 94 00 [freq BCD LSB-first] FD — 14.250000 MHz
	msg := []byte{0xFE, 0xFE, 0xE0, 0x94, 0x00, 0x00, 0x00, 0x25, 0x14, 0x00, 0xFD}
	freq, ok := parseOperatingFrequency(msg)
	if !ok {
		t.Fatal("expected frequency message")
	}
	if freq != 14250000 {
		t.Fatalf("freq=%d, want 14250000", freq)
	}

	// reply to cmd 03
	msg03 := []byte{0xFE, 0xFE, 0xE0, 0x94, 0x03, 0x00, 0x00, 0x25, 0x14, 0x00, 0xFD}
	if _, ok := parseOperatingFrequency(msg03); !ok {
		t.Fatal("expected cmd 03 frequency reply")
	}

	if _, ok := parseOperatingFrequency([]byte{0xFE, 0xFE}); ok {
		t.Fatal("short message should not parse")
	}

	// RF power echo shape must not be treated as frequency (Operate 30% → level 77)
	powerEcho := []byte{0xFE, 0xFE, 0xE0, 0x94, 0x14, 0x0A, 0x00, 0x77, 0x00, 0x00, 0xFD}
	if freq, ok := parseOperatingFrequency(powerEcho); ok {
		t.Fatalf("power echo parsed as frequency %d", freq)
	}

	// bogus "770 kHz" with cmd 00 should parse as a frame but is rejected by band logic later;
	// ensure invalid cmd still rejected
	if _, ok := parseOperatingFrequency([]byte{0xFE, 0xFE, 0xE0, 0x94, 0x14, 0x00, 0x00, 0x77, 0x00, 0x00, 0xFD}); ok {
		t.Fatal("cmd 14 should not parse as frequency")
	}
}
