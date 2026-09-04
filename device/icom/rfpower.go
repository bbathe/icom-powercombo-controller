package icom

// RFPowerToCIV converts a radio RF power percentage (0-100) to the CI-V level (0-255).
func RFPowerToCIV(percent int) int {
	t := percent * 255
	p := t / 100
	if t%100 > 0 {
		p += 1
	}
	return p
}
