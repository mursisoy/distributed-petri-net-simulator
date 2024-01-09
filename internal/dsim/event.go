package dsim

import (
	"fmt"
	"strings"
)

type Event struct {
	// Tiempo para el que debemos considerar el evento
	Clock Clock
	// A que transicion (indice transicion en subred)
	Destination TransitionId
	// Constante que mandamos
	Value Const
}

type NullMessage struct {
	Lookahead Clock
}

func (e Event) String() string {
	return fmt.Sprintf(
		"Event -> Clock:%v\tDst: %v\tValue: %v",
		e.Clock, e.Destination, e.Value)
}

type EventList []Event

func (el *EventList) insert(newEvent Event) {
	var i int // INITIALIZED to 0 !!!

	// Obtengo la posicion ordenada del evento en slice con i
	for _, e := range *el {
		if e.Clock >= newEvent.Clock {
			break
		}
		i++
	}
	*el = append((*el)[:i], append([]Event{newEvent}, (*el)[i:]...)...)
}

func (el EventList) first() Event {
	if len(el) > 0 {
		return el[0]
	}

	return Event{} //sino devuelve el tipo Event, zeroed
}

func (el *EventList) pop() Event {
	pop := Event{}
	if len(*el) > 0 {
		pop = (*el)[0]
		copy(*el, (*el)[1:])
		(*el)[len(*el)-1] = Event{} //pongo a zero el previo último Event
		(*el) = (*el)[:len(*el)-1]
	}
	return pop
}

func (el *EventList) firstEventClock() Clock {
	if len(*el) > 0 {
		return (*el)[0].Clock
	}
	return -1
}

func (el *EventList) areEventsForClock(clock Clock) bool {
	return len(*el) > 0 && (*el).firstEventClock() == clock
}

func (el EventList) String() string {
	eventList := make([]string, len(el))
	for i, e := range el {
		eventList[i] = e.String()
	}
	return strings.Join(eventList, "\n")
}
