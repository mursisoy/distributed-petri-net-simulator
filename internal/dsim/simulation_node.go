package dsim

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/mursisoy/distributed-petri-net-simulator/internal/common/clock"
	"github.com/mursisoy/distributed-petri-net-simulator/internal/common/communicator"
)

func init() {
	gob.Register(TransitionNode{})
	gob.Register(Lefs{})
	gob.Register(Event{})
	gob.Register(NullMessage{})
	gob.Register(StartSimulationRequest{})
	gob.Register(StartSimulationResponse{})
	gob.Register(PrepareSimulationRequest{})
	gob.Register(PrepareSimulationResponse{})
	gob.Register(EventRequest{})
	gob.Register(EventResponse{})
	gob.Register(NullMessageRequest{})
	gob.Register(NullMessageResponse{})
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

type SimulationNodeConfig struct {
	ListenAddress          string
	ClockLogConfig         clock.ClockLogConfig
	SimulationEngineConfig SimulationEngineConfig
}

type SimulationNode struct {
	pid                   string
	listenAddress         string
	listener              net.Listener
	done                  chan struct{}
	wg                    sync.WaitGroup
	clog                  *clock.ClockLogger
	simulationEngine      *SimulationEngine
	externalMessagesQueue chan externalMessage
	runningNodes          sync.WaitGroup
	simulationEnds        Clock
}

func NewSimulationNode(pid string, config SimulationNodeConfig) *SimulationNode {

	return &SimulationNode{
		pid:              pid,
		simulationEngine: NewSimulationEngine(config.SimulationEngineConfig),
		listenAddress:    config.ListenAddress,
		clog:             clock.NewClockLog(pid, config.ClockLogConfig),
		done:             make(chan struct{}),
		runningNodes:     sync.WaitGroup{},
	}
}

func (sn *SimulationNode) Start(ctx context.Context) (net.Addr, error) {
	var err error
	sn.listener, err = net.Listen("tcp", sn.listenAddress)
	if err != nil {
		sn.cleanup()
		return nil, fmt.Errorf("controller failed to start listener: %v", err)
	}
	sn.clog.LogInfof("Starting simulation node")
	go communicator.HandleConnections(sn.listener, sn.handleClient)
	go sn.ctxHandler(ctx)

	return sn.listener.Addr(), nil
}

func (sn *SimulationNode) handleExternalMessageQueue() {
	for message := range sn.externalMessagesQueue {
		log.Printf("Send external message: %+v", message)
		switch mt := message.payload.(type) {
		case Event:
			sn.sendExternalEvent(message.node, mt)
		case NullMessage:
			sn.sendNullMessage(message.node, mt)
		}
	}
	sn.runningNodes.Done()
}

func (sn *SimulationNode) handleClient(conn net.Conn) {
	sn.wg.Add(1)
	defer sn.wg.Done()
	defer conn.Close()

	var (
		data interface{}
		err  error
	)

	// Decode the message received or fail
	if data, err = communicator.Receive(conn); err != nil {
		sn.clog.LogErrorf("error decoding message: %s", err)
		return
	}

	// Switch between decoded messages
	switch mt := data.(type) {
	case PrepareSimulationRequest:
		sn.clog.LogMergeInfof(mt.Clock, "Prepare simulation request received")
		if !sn.simulationEngine.initialized {
			sn.externalMessagesQueue = make(chan externalMessage, 100)
			go sn.handleExternalMessageQueue()

			sn.simulationEngine.init(mt.Lefs, mt.WaitingOnSegments, mt.TransitionNodes, mt.NotificationSegments, sn.externalMessagesQueue)
			// I am a running node
			sn.runningNodes.Add(len(mt.WaitingOnSegments) + 1)
			communicator.Send(conn, PrepareSimulationResponse{Response: communicator.Response{}})
		} else {
			communicator.Send(conn, PrepareSimulationResponse{Response: communicator.Response{Error: errors.New("simulation engine already initialized")}})
		}

	case StartSimulationRequest:
		sn.clog.LogMergeInfof(mt.Clock, "Start simulation request received: %+v", mt)
		if !sn.simulationEngine.initialized {
			communicator.Send(conn, StartSimulationResponse{Response: communicator.Response{Error: errors.New("simulation engine not initialized")}})
			return
		}
		if !sn.simulationEngine.running {
			sn.simulationEnds = mt.End
			go sn.simulationEngine.simulatePeriod(0, mt.End)
			communicator.Send(conn, StartSimulationResponse{Response: communicator.Response{}})
		} else {
			communicator.Send(conn, StartSimulationResponse{Response: communicator.Response{Error: errors.New("simulation engine already running")}})
		}
	case EventRequest:
		sn.clog.LogMergeInfof(mt.Clock, "External event received: %+v", mt)
		communicator.Send(conn, EventResponse{Response: communicator.Response{}})
		sn.simulationEngine.eventFromSegment(mt.Pid) <- mt.Event
		log.Printf("Enqueued event from segment")

	case NullMessageRequest:
		sn.clog.LogMergeInfof(mt.Clock, "External null message received: %+v", mt)
		communicator.Send(conn, NullMessageResponse{Response: communicator.Response{}})
		sn.simulationEngine.nullMessageFromSegment(mt.Pid, mt.NullMessage.Lookahead)
		if mt.NullMessage.Lookahead > sn.simulationEnds {
			sn.runningNodes.Done()
		}
	default:
		sn.clog.LogErrorf("%v message type received but not handled", mt)
	}
}

func (sn *SimulationNode) ctxHandler(ctx context.Context) {

Loop:
	for {
		select {
		case <-sn.simulationEngine.done:
			sn.clog.LogInfof("Simulation engine finished")
			sn.runningNodes.Wait()
			break Loop
		case <-ctx.Done():
			break Loop
		}
	}

	sn.cleanup()
}

func (sn *SimulationNode) sendExternalEvent(node TransitionNode, event Event) {
	var (
		response interface{}
		err      error
	)
	// Prepare event request
	address := net.JoinHostPort(node.Address, node.Port)
	cc := sn.clog.LogInfof("Send event to %s: %+v", node.Name, event)
	if response, err = communicator.SendReceiveTCP(
		address,
		EventRequest{
			Request: communicator.RequestWithClock(sn.clog.GetPid(), cc),
			Event:   event,
		}); err != nil {
		log.Fatalf("send event failed to receive response from %s: %s", node.Name, err)
	}
	switch mt := response.(type) {
	case EventResponse:
		if mt.Error == nil {
			log.Printf("Received success from %v", sn.clog.GetPid())
		} else {
			log.Fatalf("received unsucessful response from %v: %s", sn.clog.GetPid(), mt.Error)
		}
	default:
		log.Fatalf("Received  unknown response from controller")
	}
}

func (sn *SimulationNode) sendNullMessage(node TransitionNode, nullMessage NullMessage) {
	var (
		response interface{}
		err      error
	)
	// Prepare event request
	address := net.JoinHostPort(node.Address, node.Port)
	cc := sn.clog.LogInfof("Send null message to %s: %+v", node.Name, nullMessage)
	if response, err = communicator.SendReceiveTCP(
		address,
		NullMessageRequest{
			Request:     communicator.RequestWithClock(sn.clog.GetPid(), cc),
			NullMessage: nullMessage,
		}); err != nil {
		log.Fatalf("send null message failed to receive response from %s", node.Name)
	}
	switch mt := response.(type) {
	case NullMessageResponse:
		if mt.Error == nil {
			log.Printf("Received success from %v", sn.clog.GetPid())
		} else {
			log.Fatalf("received unsucessful response from %v: %s", sn.clog.GetPid(), mt.Error)
		}
	default:
		log.Fatalf("Received  unknown response from controller")
	}
}

func (sn *SimulationNode) Done() <-chan struct{} {
	return sn.done
}

func (sn *SimulationNode) cleanup() {
	sn.wg.Wait()
	sn.listener.Close()
	close(sn.done)
}
