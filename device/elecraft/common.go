package elecraft

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"go.bug.st/serial"
)

const (
	readTimeout  = 333 * time.Millisecond
	responseWait = 3 * time.Second
)

var errPortClosed = errors.New("port closed")

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

// readMatchingMessage reads framed messages until match returns true or responseWait elapses.
// An empty read (read timeout with no data) is treated as no response / disconnect.
func readMatchingMessage(p serial.Port, match func(string) bool, closed func() bool) (string, error) {
	deadline := time.Now().Add(responseWait)

	for time.Now().Before(deadline) {
		if closed != nil && closed() {
			return "", errPortClosed
		}

		msg, err := readMessageFromPort(p)
		if err != nil {
			return "", err
		}
		if msg == "" {
			return "", fmt.Errorf("no serial response")
		}
		if match(msg) {
			return msg, nil
		}
	}

	return "", fmt.Errorf("serial response timeout")
}

// writeMessageToPort writes a KPA500/KAT500 formatted message to port p
func writeMessageToPort(p serial.Port, msg string) error {
	_, err := p.Write([]byte(msg))
	if err != nil {
		return err
	}

	return nil
}
