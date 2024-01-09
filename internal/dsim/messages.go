package dsim

import (
	"github.com/mursisoy/distributed-petri-net-simulator/internal/common/communicator"
)

type PrepareSimulationRequest struct {
	communicator.Request
	Lefs                 Lefs
	TransitionNodes      map[TransitionId]TransitionNode
	WaitingOnSegments    []string
	NotificationSegments []TransitionNode
}

type PrepareSimulationResponse struct {
	communicator.Response
}

type StartSimulationRequest struct {
	communicator.Request
	End Clock
}

type StartSimulationResponse struct {
	communicator.Response
}

type EventRequest struct {
	communicator.Request
	Event Event
}

type EventResponse struct {
	communicator.Response
}

type NullMessageRequest struct {
	communicator.Request
	NullMessage NullMessage
}

type NullMessageResponse struct {
	communicator.Response
}
