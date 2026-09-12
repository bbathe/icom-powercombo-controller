package status

import (
	"sync"
)

type SystemStatus int

const (
	SystemStatusRadio SystemStatus = iota
	SystemStatusKAT500
	SystemStatusKPA500

	SystemStatusLast // so we can get the number of status items defined
)

type StatusValue int

const (
	StatusUnknown StatusValue = iota
	StatusOK
	StatusFailed
)

// allow callers to register to recieve event after any status change occurs
type StatusChangeEventHandler func([]StatusValue)

var (
	mutexStatuses  sync.Mutex
	statuses       [int(SystemStatusLast)]StatusValue
	statusHandlers []StatusChangeEventHandler
)

func Attach(handler StatusChangeEventHandler) int {
	mutexStatuses.Lock()
	defer mutexStatuses.Unlock()

	statusHandlers = append(statusHandlers, handler)
	h := len(statusHandlers) - 1

	return h
}

func Detach(handle int) {
	mutexStatuses.Lock()
	defer mutexStatuses.Unlock()

	statusHandlers[handle] = nil
}

func publishTaskStatusChange() {
	snapshot := make([]StatusValue, len(statuses))
	copy(snapshot, statuses[:])

	// Call handlers synchronously so consecutive SetStatus updates keep order.
	// Handlers must not block: UI work should be posted (e.g. Synchronize).
	// Publishing with `go` raced and could apply an older snapshot last (KPA stuck gray).
	for _, h := range statusHandlers {
		if h != nil {
			h(snapshot)
		}
	}
}

func SetStatus(t SystemStatus, s StatusValue) {
	mutexStatuses.Lock()
	defer mutexStatuses.Unlock()

	if t < SystemStatusLast {
		if statuses[t] != s {
			statuses[t] = s
			publishTaskStatusChange()
		}
	}
}

func SetStatuses(s StatusValue) {
	mutexStatuses.Lock()
	defer mutexStatuses.Unlock()

	// set them all to same value
	for t := range statuses {
		statuses[t] = s
	}
	publishTaskStatusChange()
}

// Snapshot returns a copy of the current status values.
func Snapshot() []StatusValue {
	mutexStatuses.Lock()
	defer mutexStatuses.Unlock()

	out := make([]StatusValue, len(statuses))
	copy(out, statuses[:])
	return out
}
