package clock

import (
	"bytes"
	"fmt"
	"sort"
	"sync"
)

//https://github.com/alpe/messaging_spike/blob/ad68f34b5877ea8fd389482a8b5e4ad6257b6a08/vector_clock.go#L10

type ClockMap map[string]uint64

type Clock struct {
	mutex sync.RWMutex
	pid   string
	clock ClockMap
}

// VClockPayload is the data structure that is actually end on the wire
type ClockPayload struct {
	Pid   string
	Clock ClockMap
}

// NewClock returns a new, empty vector clock
func NewClock(pid string) *Clock {
	return &Clock{
		mutex: sync.RWMutex{},
		clock: ClockMap{pid: 0},
		pid:   pid,
	}
}

// FindTicks returns the clock value for a given id or false if the id is not found.
// This is a convenience function for accessing the clock value of a given id, you
// can also access the clock value directly using the map syntax, e.g.: vc["A"].
func (vc *Clock) FindTicks(id string) (uint64, bool) {
	defer vc.mutex.RUnlock()
	vc.mutex.RLock()

	ticks, ok := vc.clock[id]
	return ticks, ok
}

func (vc *Clock) copyClock() ClockMap {
	cp := make(ClockMap, len(vc.clock))
	for k, v := range vc.clock {
		cp[k] = v
	}
	return cp
}

// GetMap returns a copy of the underlying clock.
func (vc *Clock) GetClock() ClockMap {
	defer vc.mutex.RUnlock()
	vc.mutex.RLock()
	return vc.copyClock()
}

// Tick increments the clock value of the given process id by 1.
// If the process id is not found in the clock, it is added with a value of 1.
func (vc *Clock) Tick() ClockMap {
	defer vc.mutex.Unlock()
	vc.mutex.Lock()
	vc.clock[vc.pid] = vc.clock[vc.pid] + 1
	return vc.copyClock()
}

// Merge takes the maximum of all clock values in other and updates the
// values of the callee. If the callee does not contain a given id, it is
// added to the callee with the value from other.
// Merge updates the callee vector clock in place.
func (vc *Clock) Merge(other ClockMap) ClockMap {
	defer vc.mutex.Unlock()
	vc.mutex.Lock()
	for id := range other {
		if vc.clock[id] < other[id] {
			vc.clock[id] = other[id]
		}
	}
	return vc.copyClock()
}

func (vc *Clock) TickAndMerge(other ClockMap) ClockMap {
	defer vc.mutex.Unlock()
	vc.mutex.Lock()
	vc.clock[vc.pid] = vc.clock[vc.pid] + 1
	for id := range other {
		if vc.clock[id] < other[id] {
			vc.clock[id] = other[id]
		}
	}
	return vc.copyClock()
}

func (vc *Clock) String() string {
	return vc.GetClock().String()
}

func (c ClockMap) String() string {
	//sort
	ids := make([]string, len(c))
	i := 0
	for id := range c {
		ids[i] = id
		i++
	}

	sort.Strings(ids)

	var buffer bytes.Buffer
	buffer.WriteString("{")
	for i := range ids {
		buffer.WriteString(fmt.Sprintf("\"%s\":%d", ids[i], c[ids[i]]))
		if i+1 < len(ids) {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString("}")
	return buffer.String()
}
