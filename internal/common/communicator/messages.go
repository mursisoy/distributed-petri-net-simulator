package communicator

import "github.com/mursisoy/distributed-petri-net-simulator/internal/common/clock"

type Request struct {
	clock.ClockPayload
}

type Response struct {
	clock.ClockPayload
	Message string
	Error   error
}

func RequestWithClock(pid string, cc clock.ClockMap) Request {
	return Request{
		ClockPayload: clock.ClockPayload{
			Pid:   pid,
			Clock: cc,
		},
	}
}

func ResponseWithClock(pid string, cc clock.ClockMap, success bool) Response {
	return Response{
		ClockPayload: clock.ClockPayload{
			Pid:   pid,
			Clock: cc,
		},
		// Success: success,
	}
}
