package elecraft

import (
	"bytes"

	"github.com/albenik/go-serial/v2"
)

// readMessageFromPort reads a KPA500/KAT500 formatted message from port p
func readMessageFromPort(p *serial.Port) (string, error) {
	var buf bytes.Buffer
	b := []byte{0}

	for {
		n, err := p.Read(b)
		if err != nil {
			return "", err
		}

		if n > 0 {
			// accumulate message bytes
			buf.Write(b)

			// message terminator?
			if b[0] == ';' {
				// return  message
				return buf.String(), nil
			}
		}
	}
}

// writeMessageToPort writes a KPA500/KAT500 formatted message to port p
func writeMessageToPort(p *serial.Port, msg string) error {
	// write to port
	_, err := p.Write([]byte(msg))
	if err != nil {
		return err
	}

	return nil
}
