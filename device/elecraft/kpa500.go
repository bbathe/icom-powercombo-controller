package elecraft

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/bbathe/icom-powercombo-controller/util"

	"github.com/albenik/go-serial/v2"
)

type KPA500 struct {
	Port string
	Baud int

	p         *serial.Port
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
	p, err := serial.Open(port,
		serial.WithBaudrate(baud),
		serial.WithReadTimeout(333),
		serial.WithWriteTimeout(333),
	)
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

	// request power
	err := writeMessageToPort(k.p, "^WS;")
	if k.closed.IsTrue() {
		return 0, nil
	}
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}

	// read response from kpa500
	for {
		msg, err := readMessageFromPort(k.p)
		if k.closed.IsTrue() {
			return 0, nil
		}
		if err != nil {
			log.Printf("%+v", err)
			return 0, err
		}
		if msg == "" {
			// no response, kpa500 disconnected?
			return 0, nil
		}

		// our response?
		if strings.HasPrefix(msg, "^WS") {
			// RSP format: ^WSppp sss;
			s := strings.TrimPrefix(msg, "^WS")
			s = strings.TrimSuffix(s, ";")

			if len(s) == 0 {
				// no response, kpa500 disconnected?
				return 0, nil
			}

			ss := strings.Split(s, " ")
			w := ss[0]

			// convert to number
			watts, err := strconv.Atoi(w)
			if err != nil {
				log.Printf("%+v", err)
				return 0, err
			}

			return watts, nil
		}
	}
}

// GetFault gets the current fault identifier from the KPA500, zero indicates no faults are active
func (k *KPA500) GetFault() (int, error) {
	k.mutexPort.Lock()
	defer k.mutexPort.Unlock()

	// request current fault
	err := writeMessageToPort(k.p, "^FL;")
	if k.closed.IsTrue() {
		return 0, nil
	}
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}

	// read response from kpa500
	for {
		msg, err := readMessageFromPort(k.p)
		if k.closed.IsTrue() {
			return 0, nil
		}
		if err != nil {
			log.Printf("%+v", err)
			return 0, err
		}
		if msg == "" {
			// no response, kpa500 disconnected?
			return 255, nil
		}

		// our response?
		if strings.HasPrefix(msg, "^FL") {
			// RSP format: ^FLnn;
			s := strings.TrimPrefix(msg, "^FL")
			s = strings.TrimSuffix(s, ";")

			if len(s) == 0 {
				// no response, kpa500 disconnected?
				return 255, nil
			}

			// convert to number
			fault, err := strconv.Atoi(s)
			if err != nil {
				log.Printf("%+v", err)
				return 0, err
			}

			return fault, nil
		}
	}
}
