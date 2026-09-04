package icom

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/bbathe/icom-powercombo-controller/util"

	"go.bug.st/serial"
)

type Radio struct {
	Port    string
	Baud    int
	Address string

	p         serial.Port
	mutexPort sync.Mutex
	f         bool
	closed    util.AtomFlag
}

var (
	errPortClosed = fmt.Errorf("port closed")
)

const (
	readTimeout  = 333 * time.Millisecond
	responseWait = 3 * time.Second
)

// OpenRadio creates a connection with the radio
func OpenRadio(port string, baud int, address string) (*Radio, error) {
	p, err := openSerialPort(port, baud)
	if err != nil {
		log.Printf("%+v", err)
		return nil, err
	}

	r := new(Radio)
	r.Port = port
	r.Baud = baud
	r.Address = address
	r.p = p

	return r, nil
}

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

// Close closes the connection with the radio
func (r *Radio) Close() error {
	r.closed.Set(true)

	r.mutexPort.Lock()
	defer r.mutexPort.Unlock()

	return r.p.Close()
}

// readCIVMessageFromPort reads bytes from port and returns CIV message
func (r *Radio) readCIVMessageFromPort() ([]byte, error) {
	var buf bytes.Buffer
	b := []byte{0}

	for {
		n, err := r.p.Read(b)
		if r.closed.IsTrue() {
			return []byte{}, errPortClosed
		}
		if err != nil {
			log.Printf("%+v", err)
			return []byte{}, err
		}

		if n > 0 {
			// accumulate message bytes
			buf.Write(b)

			// message terminator?
			if b[0] == 0xFD {
				// return CIV message
				return buf.Bytes(), nil
			}
		} else {
			break
		}
	}

	// return message
	return buf.Bytes(), nil
}

// writeCIVMessageToPort write byte equalivalent of msg to port
func (r *Radio) writeCIVMessageToPort(msg string) error {
	// convert from hex string to bytes
	b, err := hex.DecodeString(msg)
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	// write to port
	_, err = r.p.Write(b)
	if r.closed.IsTrue() {
		return errPortClosed
	}
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	return nil
}

// QueryFrequency sends CI-V command 03 and waits up to responseWait for a frequency reply.
// Returns -1, nil when the radio does not answer (powered off / unplugged CI-V).
func (r *Radio) QueryFrequency() (int64, error) {
	r.mutexPort.Lock()
	defer r.mutexPort.Unlock()

	err := r.writeCIVMessageToPort(fmt.Sprintf("FEFE%sE003FD", r.Address))
	if err != nil {
		if err == errPortClosed {
			return -1, nil
		}
		log.Printf("%+v", err)
		return 0, err
	}
	r.f = true

	deadline := time.Now().Add(responseWait)
	for time.Now().Before(deadline) {
		msg, err := r.readCIVMessageFromPort()
		if err != nil {
			if err == errPortClosed {
				return -1, nil
			}
			log.Printf("%+v", err)
			return 0, err
		}
		if freq, ok := parseOperatingFrequency(msg); ok {
			return freq, nil
		}
	}
	return -1, nil
}

// GetFrequency returns the current radio frequency
// it does this by polling for the "Transfer operating frequency data" broadcast message
// if port is closed during reading, -1 is returned
func (r *Radio) GetFrequency() (int64, error) {
	r.mutexPort.Lock()
	defer r.mutexPort.Unlock()

	if !r.f {
		// first time after connecting to radio, query for frequency
		err := r.writeCIVMessageToPort(fmt.Sprintf("FEFE%sE003FD", r.Address))
		if err != nil {
			if err == errPortClosed {
				return -1, nil
			}
			log.Printf("%+v", err)
			return 0, err
		}

		r.f = true
	}

	// read ci-v message
	msg, err := r.readCIVMessageFromPort()
	if err != nil {
		if err == errPortClosed {
			return -1, nil
		}
		log.Printf("%+v", err)
		return 0, err
	}

	if freq, ok := parseOperatingFrequency(msg); ok {
		return freq, nil
	}

	return -1, nil
}

func parseOperatingFrequency(msg []byte) (int64, bool) {
	// is it operating frequency data?
	if len(msg) != 11 || (msg[2] != 0xE0 && msg[2] != 0x00) {
		return 0, false
	}

	// radio sends as least significant byte first, flip order of bytes
	fd := fmt.Sprintf("%02X%02X%02X%02X%02X", msg[9], msg[8], msg[7], msg[6], msg[5])

	freq, err := strconv.ParseInt(fd, 10, 64)
	if err != nil {
		return 0, false
	}
	return freq, true
}

// SetRFPower sets the RF Power of the radio
func (r *Radio) SetRFPower(power int) error {
	r.mutexPort.Lock()
	defer r.mutexPort.Unlock()

	// calculate radio power setting from percentage
	p := RFPowerToCIV(power)

	// set rf power
	err := r.writeCIVMessageToPort(fmt.Sprintf("FEFE%sE0140A%04dFD", r.Address, p))
	if err != nil {
		if err == errPortClosed {
			return nil
		}
		log.Printf("%+v", err)
		return err
	}

	deadline := time.Now().Add(responseWait)
	for time.Now().Before(deadline) {
		msg, err := r.readCIVMessageFromPort()
		if err != nil {
			if err == errPortClosed {
				return nil
			}
			log.Printf("%+v", err)
			return err
		}

		// response for us from radio?
		if len(msg) == 6 && msg[2] == 0xE0 {
			// check status returned from radio
			if msg[4] != 0xFB {
				err = fmt.Errorf("error response from radio")
				log.Printf("%+v", err)
				return err
			}
			return nil
		}
	}

	err = fmt.Errorf("serial response timeout")
	log.Printf("%+v", err)
	return err
}
