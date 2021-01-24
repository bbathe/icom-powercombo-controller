package data

import (
	"sync"
)

type Radio struct {
	Frequency int64
	Band      int
}

type KPA500 struct {
	Mode  int
	Power int
}

type Data struct {
	Radio
	KPA500
}

// allow callers to register to recieve event after any change occurs
type DataChangeEventHandler func(Data)

var (
	mutexData sync.Mutex
	radio     Radio
	kpa500    KPA500

	dataChangedHandlers []DataChangeEventHandler
)

func Attach(handler DataChangeEventHandler) int {
	mutexData.Lock()
	defer mutexData.Unlock()

	dataChangedHandlers = append(dataChangedHandlers, handler)
	h := len(dataChangedHandlers) - 1

	return h
}

func Detach(handle int) {
	mutexData.Lock()
	defer mutexData.Unlock()

	dataChangedHandlers[handle] = nil
}

func publishDataChange() {
	for _, h := range dataChangedHandlers {
		if h != nil {
			go h(Data{
				Radio:  radio,
				KPA500: kpa500,
			})
		}
	}
}

// update shared state about the Radio
// pass -1 for any data that shouldn't be updated
func (rd Radio) Update() {
	mutexData.Lock()
	defer mutexData.Unlock()

	if rd.Frequency > -1 {
		radio.Frequency = rd.Frequency
	}
	if rd.Band > -1 {
		radio.Band = rd.Band
	}

	publishDataChange()
}

// update shared state about the KPA500
// pass -1 for any data that shouldn't be updated
func (kd KPA500) Update() {
	mutexData.Lock()
	defer mutexData.Unlock()

	if kd.Mode > -1 {
		kpa500.Mode = kd.Mode
	}
	if kd.Power > -1 {
		kpa500.Power = kd.Power
	}

	publishDataChange()
}

// GetRadioData returns a consistent copy of the current Radio shared state
func GetRadioData() Radio {
	mutexData.Lock()
	defer mutexData.Unlock()

	r := radio
	return r
}

// GetKPA500Data returns a consistent copy of the current KPA500 shared state
func GetKPA500Data() KPA500 {
	mutexData.Lock()
	defer mutexData.Unlock()

	kpa := kpa500
	return kpa
}
