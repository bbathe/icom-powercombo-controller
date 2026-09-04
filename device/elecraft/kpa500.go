package elecraft

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/bbathe/icom-powercombo-controller/util"

	"go.bug.st/serial"
)

type KPA500 struct {
	Port string
	Baud int

	p         serial.Port
	mutexPort sync.Mutex
	closed    util.AtomFlag
}

var (
	// from the KPA500 documentation
	bandLookup = map[int]string{
		160: "00",
		80:  "01",
		60:  "02",
		40:  "03",
		30:  "04",
		20:  "05",
		17:  "06",
		15:  "07",
		12:  "08",
		10:  "09",
		6:   "10",
	}
)

// OpenKPA500 creates a connection with the KPA500
func OpenKPA500(port string, baud int) (*KPA500, error) {
	p, err := openSerialPort(port, baud)
	if err != nil {
		log.Printf("%+v", err)
		return nil, err
	}

	k := new(KPA500)
	k.p = p
	k.Port = port
	k.Baud = baud

	return k, nil
}

// Close closes the connection with the KPA500
func (k *KPA500) Close() error {
	k.closed.Set(true)

	k.mutexPort.Lock()
	defer k.mutexPort.Unlock()

	return k.p.Close()
}

// SetMode sets the operate/standby mode of the KPA500
func (k *KPA500) SetMode(mode int) error {
	k.mutexPort.Lock()
	defer k.mutexPort.Unlock()

	err := writeMessageToPort(k.p, fmt.Sprintf("^OS%d;", mode))
	if k.closed.IsTrue() {
		return nil
	}
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	return nil
}

// SetBand sets the current band on the KPA500
func (k *KPA500) SetBand(band int) error {
	k.mutexPort.Lock()
	defer k.mutexPort.Unlock()

	err := writeMessageToPort(k.p, fmt.Sprintf("^BN%s;", bandLookup[band]))
	if k.closed.IsTrue() {
		return nil
	}
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	return nil
}

// GetPower gets the current output power (in watts) from the KPA500
func (k *KPA500) GetPower() (int, error) {
	k.mutexPort.Lock()
	defer k.mutexPort.Unlock()

	err := writeMessageToPort(k.p, "^WS;")
	if k.closed.IsTrue() {
		return 0, nil
	}
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}

	msg, err := readMatchingMessage(k.p, func(msg string) bool {
		return strings.HasPrefix(msg, "^WS")
	}, k.closed.IsTrue)
	if errors.Is(err, errPortClosed) {
		return 0, nil
	}
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}

	watts, err := parseWS(msg)
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}

	return watts, nil
}

// GetFault gets the current fault identifier from the KPA500, zero indicates no faults are active
func (k *KPA500) GetFault() (int, error) {
	k.mutexPort.Lock()
	defer k.mutexPort.Unlock()

	err := writeMessageToPort(k.p, "^FL;")
	if k.closed.IsTrue() {
		return 0, nil
	}
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}

	msg, err := readMatchingMessage(k.p, func(msg string) bool {
		return strings.HasPrefix(msg, "^FL")
	}, k.closed.IsTrue)
	if errors.Is(err, errPortClosed) {
		return 0, nil
	}
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}

	fault, err := parseFL(msg)
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}

	return fault, nil
}

// GetPAVoltsCurrent gets the PA Voltage and Current from the KPA500
func (k *KPA500) GetPAVoltsCurrent() (float64, float64, error) {
	k.mutexPort.Lock()
	defer k.mutexPort.Unlock()

	err := writeMessageToPort(k.p, "^VI;")
	if k.closed.IsTrue() {
		return 0.0, 0.0, nil
	}
	if err != nil {
		log.Printf("%+v", err)
		return 0.0, 0.0, err
	}

	msg, err := readMatchingMessage(k.p, func(msg string) bool {
		return strings.HasPrefix(msg, "^VI")
	}, k.closed.IsTrue)
	if errors.Is(err, errPortClosed) {
		return 0.0, 0.0, nil
	}
	if err != nil {
		log.Printf("%+v", err)
		return 0.0, 0.0, err
	}

	volts, amps, err := parseVI(msg)
	if err != nil {
		log.Printf("%+v", err)
		return 0.0, 0.0, err
	}

	return volts, amps, nil
}
