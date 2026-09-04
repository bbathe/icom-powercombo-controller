package elecraft

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/bbathe/icom-powercombo-controller/util"

	"go.bug.st/serial"
)

type KAT500 struct {
	Port string
	Baud int

	p         serial.Port
	mutexPort sync.Mutex
	closed    util.AtomFlag
}

// OpenKAT500 creates a connection with the KAT500
func OpenKAT500(port string, baud int) (*KAT500, error) {
	p, err := openSerialPort(port, baud)
	if err != nil {
		log.Printf("%+v", err)
		return nil, err
	}

	k := new(KAT500)
	k.p = p
	k.Port = port
	k.Baud = baud

	return k, nil
}

// Close closes the connection with the KAT500
func (k *KAT500) Close() error {
	k.closed.Set(true)

	k.mutexPort.Lock()
	defer k.mutexPort.Unlock()

	return k.p.Close()
}

// SetFrequency sets the current frequency on the KAT500
func (k *KAT500) SetFrequency(freq int64) error {
	k.mutexPort.Lock()
	defer k.mutexPort.Unlock()

	err := writeMessageToPort(k.p, fmt.Sprintf("F %d;", freq/1000))
	if k.closed.IsTrue() {
		return nil
	}
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	return nil
}

// GetFault gets the current fault identifier from the KAT500, zero indicates no faults are active
func (k *KAT500) GetFault() (int, error) {
	k.mutexPort.Lock()
	defer k.mutexPort.Unlock()

	err := writeMessageToPort(k.p, "FLT;")
	if k.closed.IsTrue() {
		return 0, nil
	}
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}

	msg, err := readMatchingMessage(k.p, func(msg string) bool {
		return strings.HasPrefix(msg, "FLT")
	}, k.closed.IsTrue)
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}
	if msg == "" {
		return 0, nil
	}

	s := strings.TrimPrefix(msg, "FLT")
	s = strings.TrimSuffix(s, ";")
	if len(s) == 0 {
		return 0, fmt.Errorf("no serial response")
	}

	fault, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}

	return fault, nil
}

// GetVSWR gets the current Voltage Standing Wave Ratio from the KAT500
func (k *KAT500) GetVSWR() (float64, error) {
	k.mutexPort.Lock()
	defer k.mutexPort.Unlock()

	err := writeMessageToPort(k.p, "VSWR;")
	if k.closed.IsTrue() {
		return 0, nil
	}
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}

	msg, err := readMatchingMessage(k.p, func(msg string) bool {
		return strings.HasPrefix(msg, "VSWR")
	}, k.closed.IsTrue)
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}
	if msg == "" {
		return 0, nil
	}

	s := strings.TrimPrefix(msg, "VSWR ")
	s = strings.TrimSuffix(s, ";")
	if len(s) == 0 {
		return 0, fmt.Errorf("no serial response")
	}

	vswr, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}

	return vswr, nil
}

func (k *KAT500) FullTune() error {
	k.mutexPort.Lock()
	defer k.mutexPort.Unlock()

	err := writeMessageToPort(k.p, "T;")
	if k.closed.IsTrue() {
		return nil
	}
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	msg, err := readMatchingMessage(k.p, func(msg string) bool {
		return msg == "FT;"
	}, k.closed.IsTrue)
	if err != nil {
		log.Printf("%+v", err)
		return err
	}
	if msg == "" {
		return nil
	}

	return nil
}
