package dsim

import (
	"fmt"
	"log"
	"time"
)

type Clock float32

type externalMessage struct {
	node    TransitionNode
	payload interface{}
}

// ResultadoTransition holds fired transition id and time of firing
type TransitionResult struct {
	TransitionId      TransitionId
	ClockTriggerValue Clock
}

type SimulationEngineConfig struct {
	Lookahead  Clock
	ResultPath string
}

type TransitionNode struct {
	Name    string
	Address string
	Port    string
}

type SegmentLink struct {
	clock      Clock
	eventQueue chan Event
	lookahead  chan Clock
	eventList  EventList
}

// SimulationEngine is the basic data type for simulation execution
type SimulationEngine struct {
	clock                 Clock // Valor de mi reloj local
	lookahead             Clock
	lefs                  Lefs // Estructura de datos del simulador
	externalEventList     EventList
	eventList             EventList // Lista de eventos a procesar
	transitionResults     []TransitionResult
	eventNumber           float64 // cantidad de eventos ejecutados
	waitingOnSegments     map[string]*SegmentLink
	notificationSegments  []TransitionNode
	transitionNodes       map[TransitionId]TransitionNode
	initialized           bool
	running               bool
	externalMessagesQueue chan<- externalMessage
	done                  chan struct{}
}

func NewSimulationEngine(sec SimulationEngineConfig) *SimulationEngine {
	return &SimulationEngine{
		lookahead:   sec.Lookahead,
		initialized: false,
		running:     false,
		done:        make(chan struct{}),
	}
}

func (se *SimulationEngine) init(lefs Lefs, waitingOnSegments []string, transitionNodes map[TransitionId]TransitionNode, notificationSegments []TransitionNode, externalMessagesQueue chan<- externalMessage) {
	se.lefs = lefs
	se.transitionNodes = transitionNodes
	se.externalEventList = make(EventList, 0, 100)
	se.eventList = make(EventList, 0, 100)
	se.transitionResults = make([]TransitionResult, 0, 100)
	se.eventNumber = 0
	se.waitingOnSegments = make(map[string]*SegmentLink)
	for _, v := range waitingOnSegments {
		segmentLink := SegmentLink{0, make(chan Event, 100), make(chan Clock, 1), make(EventList, 0, 100)}
		se.waitingOnSegments[v] = &segmentLink
	}
	se.notificationSegments = notificationSegments
	se.externalMessagesQueue = externalMessagesQueue
	log.Println("Initialized simulation engine")
	log.Printf("%+v", se)
	se.initialized = true
}

func (se *SimulationEngine) eventFromSegment(id string) chan<- Event {
	return se.waitingOnSegments[id].eventQueue
}

func (se *SimulationEngine) nullMessageFromSegment(id string, lookahead Clock) {
	select {
	case <-se.waitingOnSegments[id].lookahead:
		se.waitingOnSegments[id].lookahead <- lookahead
	default:
		se.waitingOnSegments[id].lookahead <- lookahead
	}
}

func (se *SimulationEngine) fireTransition(tId TransitionId) {
	// Prepare 5 local variables
	tl := se.lefs.Network
	t := tl[tId]

	// First apply Iul propagations (Inmediate : 0 propagation time)
	for _, trCo := range t.Update {
		tl[trCo.TransitionId].updateFuncValue(trCo.Constant)
	}

	for _, trCo := range t.Propagate {
		if trCo.TransitionId < 0 {
			se.externalEventList.insert(Event{t.Clock + t.Duration,
				trCo.TransitionId,
				trCo.Constant})
		} else {
			// tiempo = tiempo de la transicion + coste disparo
			se.eventList.insert(Event{t.Clock + t.Duration,
				trCo.TransitionId,
				trCo.Constant})
		}
	}
}

func (se *SimulationEngine) fireEnabledTransitions() {
	for !se.lefs.Sensitized.isEmpty() { //while
		tId := se.lefs.getSensitized()
		se.fireTransition(tId)
		se.transitionResults = append(se.transitionResults,
			TransitionResult{tId, se.clock})
	}
}

func (se *SimulationEngine) forwardTime() Clock {
	var lowerBoundClock Clock

	// Initial min time is first event clock
	if lowerBoundClock = se.eventList.firstEventClock(); lowerBoundClock == -1 {
		lowerBoundClock = se.clock
	}

	// If any waiting segment has a lower clock set as minTime
	for _, v := range se.waitingOnSegments {
		if v.clock < lowerBoundClock {
			lowerBoundClock = v.clock
		}
	}

	// Wait for the lowest clock segments wither by event, or by lookahead
	for _, v := range se.waitingOnSegments {
		if v.clock == lowerBoundClock {
			select {
			case clock := <-v.lookahead:
				v.clock = clock
			case event := <-v.eventQueue:
				se.eventList.insert(event)
			}
		}
	Loop:
		for {
			select {
			case event := <-v.eventQueue:
				se.eventList.insert(event)
			default:
				break Loop
			}
		}
	}

	if lowerBoundClock = se.eventList.firstEventClock(); lowerBoundClock == -1 {
		lowerBoundClock = se.clock + se.lookahead
	}

	return lowerBoundClock
}

func (se *SimulationEngine) handleEvents() {
	var event Event
	tl := se.lefs.Network // obtener lista de transiciones de Lefs
	for se.eventList.areEventsForClock(se.clock) {
		event = se.eventList.pop() // extraer evento más reciente
		transitionId := getLocalTransitionId(event.Destination)
		log.Printf("handleEvents: %s: %s", tl[transitionId], event)
		// Establecer nuevo valor de la funcion
		tl[transitionId].updateFuncValue(event.Value)
		// Establecer nuevo valor del tiempo
		tl[transitionId].updateClock(event.Clock)
		se.eventNumber++
	}

}

func (se *SimulationEngine) simulateStep(End Clock) {
	se.lefs.updateSensitized(se.clock)
	log.Printf("Sensitized: \n%+v", se.lefs.Sensitized)

	// Fire enabled transitions and produce events
	se.fireEnabledTransitions()
	log.Printf("Events: \n%s", se.eventList)
	log.Printf("ExternalEvents: \n%s", se.externalEventList)

	se.sendExternalEvents()

	// advance local clock to soonest available event
	se.clock = se.forwardTime()

	log.Printf("Clock: %v", se.clock)

	// if events exist for current local clock, process them
	se.handleEvents()
}

func (se *SimulationEngine) sendExternalEvents() {
	// Create a map to notify nodes
	notificationSegments := make(map[string]TransitionNode, len(se.notificationSegments))
	for _, v := range se.notificationSegments {
		notificationSegments[v.Name] = v
	}

	// Send events
	for len(se.externalEventList) > 0 {
		event := se.externalEventList.pop()
		node := se.getTransitionNode(event.Destination)
		log.Printf("%+v", node)
		delete(notificationSegments, node.Name)
		se.externalMessagesQueue <- externalMessage{node, event}
	}

	// Send null messages to not evented nodes
	for _, node := range notificationSegments {
		se.externalMessagesQueue <- externalMessage{node, NullMessage{Lookahead: se.clock + se.lookahead}}
	}
}

func getLocalTransitionId(id TransitionId) TransitionId {
	if id < 0 {
		return (1 + id) * -1
	} else {
		return id
	}
}

func (se *SimulationEngine) getTransitionNode(id TransitionId) TransitionNode {
	return se.transitionNodes[getLocalTransitionId(id)]
}

func (se *SimulationEngine) simulatePeriod(Start Clock, End Clock) {
	se.running = true
	begin := time.Now()

	// Inicializamos el reloj local
	// ------------------------------------------------------------------
	se.clock = Start

	for se.clock < End {
		se.simulateStep(End)
	}

	elapsedTime := time.Since(begin)

	fmt.Printf("Eventos por segundo = %f",
		se.eventNumber/elapsedTime.Seconds())

	fmt.Printf("Transition results\n")
	fmt.Printf("==================\n")
	for _, tr := range se.transitionResults {
		fmt.Printf("%+v\n", tr)
	}
	se.running = false
	for _, node := range se.notificationSegments {
		se.externalMessagesQueue <- externalMessage{node, NullMessage{Lookahead: End + se.lookahead}}
	}
	close(se.externalMessagesQueue)
	close(se.done)
}
