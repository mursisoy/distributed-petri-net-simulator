// Este programa requiere 2 parámetros de entrada :
//   - Nombre fichero json de Lefs
//   - Número de ciclo final
//
// Ejemplo : censim  testdata/PrimerEjemplo.rdp.subred0.json  5
package main

import (
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	// "github.com/mursisoy/simuladores/dsim"
	// "github.com/mursisoy/simuladores/internal/clock"
	// "github.com/mursisoy/simuladores/internal/common"
	"github.com/mursisoy/distributed-petri-net-simulator/internal/common/clock"
	"github.com/mursisoy/distributed-petri-net-simulator/internal/common/communicator"
	"github.com/mursisoy/distributed-petri-net-simulator/internal/dsim"
	"golang.org/x/crypto/ssh"
)

type Node struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    string `json:"port"`
}

func init() {
	gob.Register(dsim.TransitionNode{})
	gob.Register(dsim.Lefs{})
	gob.Register(dsim.StartSimulationRequest{})
	gob.Register(dsim.StartSimulationResponse{})
	gob.Register(dsim.PrepareSimulationRequest{})
	gob.Register(dsim.PrepareSimulationResponse{})
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

var clog *clock.ClockLogger

func main() {
	// Create a channel to receive signals.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	logsDir := fmt.Sprintf("%s/dsim/logs", home)
	resultsDir := fmt.Sprintf("%s/dsim/results", home)

	os.MkdirAll(logsDir, os.ModePerm)
	os.MkdirAll(resultsDir, os.ModePerm)

	clog = clock.NewClockLog("dsl", clock.ClockLogConfig{
		Priority:    clock.DEBUG,
		FileOutput:  true,
		LogFilename: fmt.Sprintf("%s/dsim-launcher.log", logsDir),
	})

	// cargamos un fichero de estructura Lef en formato json para centralizado
	// os.Args[0] es el nombre del programa que no nos interesa

	var nodeFile string
	flag.StringVar(&nodeFile, "nodeFile", "simulation-nodes.json", "The simulation nodes list")

	var nodeCmd string
	flag.StringVar(&nodeCmd, "nodeCmd", "", "The simulation node exec")

	var period int
	flag.IntVar(&period, "period", 10, "The simulation period")

	// Enable command-line parsing
	flag.Parse()
	args := flag.Args()

	nodeList := loadNodesFromFile(nodeFile)

	lefList, _ := loadLefs(args[0], nodeList)

	// fmt.Println(nodeList)
	// fmt.Println(lefList)

	if len(nodeList) < len(lefList) {
		log.Fatal("Not enough nodes in node list")
	}

	user, err := user.Current()
	if err != nil {
		log.Fatalf(err.Error())
	}
	sshConfig := &ssh.ClientConfig{
		User: user.Username,
		Auth: []ssh.AuthMethod{
			SSHAgent(),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			log.Printf("hostkey: %q %v %v\n", hostname, remote, key)
			return nil
		},
	}

	var wg sync.WaitGroup
	var sshSessions []*ssh.Session

	// Goroutine to catch shutdown signals
	go func() {
		sig := <-sigCh
		fmt.Printf("Received signal: %v\n", sig)
		for _, session := range sshSessions {
			session.Signal(ssh.SIGINT)
		}
		os.Exit(1)
	}()

	// Create a node map to detect duplicates and iterate in random way
	nodeMap := make(map[string]Node, len(nodeList))
	for _, node := range nodeList {
		if _, ok := nodeMap[net.JoinHostPort(node.Address, node.Port)]; ok {
			log.Printf("Warning: Duplicate simulation node service detected")
		} else {
			nodeMap[net.JoinHostPort(node.Address, node.Port)] = node
		}
	}

	var simulationNodes []Node

	// Launch simulation nodes through ssh
	for address, node := range nodeMap {

		client := &SSHClient{
			Config: sshConfig,
			Host:   node.Address,
			Port:   22,
		}

		// Create a log for every session
		var f io.Writer
		f, _ = os.Create(fmt.Sprintf("%s/ssh-%s.log", logsDir, node.Name))

		// Create ssh command
		cmd := &SSHCommand{
			Path: fmt.Sprintf("%s -listen %s -id %s -resultpath %s/%s.txt -logfile %s/%s.log", nodeCmd, address, node.Name, resultsDir, node.Name, logsDir, node.Name),
			// Env:    []string{"LC_DIR=/"},
			Stdin:  os.Stdin,
			Stdout: f,
			Stderr: f,
		}

		fmt.Printf("Running command: %s\n", cmd.Path)

		var (
			session *ssh.Session
			err     error
		)

		if session, err = client.newSession(); err != nil {
			log.Printf("Cannot open session to %v", client)
			continue
		}

		wg.Add(1)
		go func() {
			defer session.Close()
			defer wg.Done()
			if err := client.RunCommand(session, cmd); err != nil {
				fmt.Fprintf(os.Stderr, "command run error: %s\n", err)
				sigCh <- syscall.SIGTERM
			}
		}()

		if checkSimulationNode(node) == nil {
			log.Printf("Node check %v", node)
			sshSessions = append(sshSessions, session)
			simulationNodes = append(simulationNodes, node)
		} else {
			session.Close()
			continue
		}

		if len(simulationNodes) == len(lefList) {
			break
		}
	}

	// Got matchin nodes
	if len(simulationNodes) == len(lefList) {
		// Map node address to transition
		transitionNodes := createTransitionNodeMap(lefList, simulationNodes)
		nodesToFrom := make(map[string][]string)
		nodesFromTo := make(map[string][]dsim.TransitionNode)
		for _, l := range lefList {
			for _, n := range l.Network {
				if n.External {
					for _, t := range n.Propagate {
						propagateNode := transitionNodes[(1+t.TransitionId)*-1]
						localNode := transitionNodes[n.Id]
						nodesToFrom[propagateNode.Name] = append(nodesToFrom[propagateNode.Name], localNode.Name)
						nodesFromTo[localNode.Name] = append(nodesFromTo[localNode.Name], propagateNode)
					}
				}
			}
		}

		for i, node := range simulationNodes {
			sendNetworkToNode(node, lefList[i], transitionNodes, nodesToFrom[node.Name], nodesFromTo[node.Name])
		}
		launchSimulation(simulationNodes, period)

		wg.Wait()
	} else {
		sigCh <- syscall.SIGINT
		log.Print(fmt.Errorf("not enough nodes"))
	}
}

func loadLefs(networkFilesLocation string, nodeList []Node) ([]dsim.Lefs, map[dsim.TransitionId]int) {

	matches, err := filepath.Glob(networkFilesLocation + ".subred*.json")

	if err != nil {
		log.Fatal(err)
	}

	if len(matches) == 0 {
		log.Fatalf("No subnetworks found")
	}

	// Map with networks
	networkList := make([]dsim.Lefs, len(matches))

	// Map with global ids and
	transitionLefMap := make(map[dsim.TransitionId]int)

	for i, networkFile := range matches {
		lef, _ := dsim.Load(networkFile)
		networkList[i] = lef
		for _, v := range lef.Network {
			transitionLefMap[v.Id] = i
		}
	}
	return networkList, transitionLefMap
}

func loadNodesFromFile(nodeFile string) []Node {
	var (
		nodeList []Node
		file     []byte
		err      error
	)
	if file, err = os.ReadFile(nodeFile); err != nil {
		log.Fatalf(err.Error())
	}
	_ = json.Unmarshal([]byte(file), &nodeList)
	return nodeList
}

func launchSimulation(simulationNodes []Node, period int) {

	for _, v := range simulationNodes {

		address := net.JoinHostPort(v.Address, v.Port)
		cc := clog.LogInfof("Send start simulation request to %s", address)
		response, _ := communicator.SendReceiveTCP(address, dsim.StartSimulationRequest{
			Request: communicator.RequestWithClock(clog.GetPid(), cc),
			End:     dsim.Clock(period),
		})

		switch mt := response.(type) {
		case dsim.StartSimulationResponse:
			if mt.Error == nil {
				log.Printf("Received success from %v", v.Name)
			} else {
				log.Fatalf("Received unsucessful response from %v: %s", v.Name, mt.Error)
			}
		default:
			log.Fatalf("Received  unknown response from controller")
		}
	}
}

func sendNetworkToNode(node Node, lef dsim.Lefs, transitionNodes map[dsim.TransitionId]dsim.TransitionNode, waitingOnSegments []string, notificationSegments []dsim.TransitionNode) {
	address := net.JoinHostPort(node.Address, node.Port)
	cc := clog.LogInfof("Send prepare simulation request to %s", address)
	response, _ := communicator.SendReceiveTCP(address,
		dsim.PrepareSimulationRequest{
			Request:              communicator.RequestWithClock(clog.GetPid(), cc),
			Lefs:                 lef,
			TransitionNodes:      transitionNodes,
			WaitingOnSegments:    waitingOnSegments,
			NotificationSegments: notificationSegments,
		})

	switch mt := response.(type) {
	case dsim.PrepareSimulationResponse:
		if mt.Error == nil {
			log.Printf("Received success from %v", node.Name)
		} else {
			log.Fatalf("Received unsucessful response from %v: %s", node.Name, mt.Error)
		}
	default:
		log.Fatalf("Received  unknown response from controller :%+v", mt)
	}
}

func createTransitionNodeMap(lefs []dsim.Lefs, simulationNodes []Node) map[dsim.TransitionId]dsim.TransitionNode {
	transitionNodeMap := make(map[dsim.TransitionId]dsim.TransitionNode)
	for i, l := range lefs {
		for _, n := range l.Network {
			transitionNodeMap[n.Id] = dsim.TransitionNode(simulationNodes[i])
		}
	}
	return transitionNodeMap
}

func checkSimulationNode(node Node) error {
	timeToSleep := time.Duration(500) * time.Millisecond
	timeout := 2 * time.Second
	i := 0
	for {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(node.Address, node.Port), timeout)
		if conn != nil {
			conn.Close()
			return nil
		}
		if i < 5 {
			time.Sleep(timeToSleep)
			timeToSleep *= 2
			i++
		} else {
			return err
		}
	}
}
