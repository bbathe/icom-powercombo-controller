package elecraft

import (
	"bytes"
	"time"

	"go.bug.st/serial"
)

const readTimeout = 333 * time.Millisecond

func openSerialPort(name string, baud int) (serial.Port, error) {
	p, err := serial.Open(name, &serial.Mode{BaudRate: baud})
	if err != nil {
		return nil, err
	}
	if err := p.SetReadTimeout(readTimeout); err != nil {
		_ = p.Close()
		return nil, err
	}
	return p, nil
}

// readMessageFromPort reads a KPA500/KAT500 formatted message from port p
func readMessageFromPort(p serial.Port) (string, error) {
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
				break
			}
		} else {
			break
		}
	}

	// return message
	return buf.String(), nil
}

// writeMessageToPort writes a KPA500/KAT500 formatted message to port p
func writeMessageToPort(p serial.Port, msg string) error {
	_, err := p.Write([]byte(msg))
	if err != nil {
		return err
	}

	return nil
}
