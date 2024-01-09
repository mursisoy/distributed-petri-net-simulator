// Este programa requiere 2 parámetros de entrada :
//   - Nombre fichero json de Lefs
//   - Número de ciclo final
//
// Ejemplo : censim  testdata/PrimerEjemplo.rdp.subred0.json  5
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mursisoy/distributed-petri-net-simulator/internal/common/clock"
	"github.com/mursisoy/distributed-petri-net-simulator/internal/dsim"
)

func main() {
	// Create a channel to receive signals.
	sigCh := make(chan os.Signal, 1)
	// cargamos un fichero de estructura Lef en formato json para centralizado
	// os.Args[0] es el nombre del programa que no nos interesa

	var listenAddress, id, logfile, resultPath string
	flag.StringVar(&listenAddress, "listen", ":0", "The worker listen port")

	flag.StringVar(&id, "id", listenAddress, "The worker id")

	flag.StringVar(&resultPath, "resultpath", "", "The worker id")

	flag.StringVar(&logfile, "logfile", fmt.Sprintf("/tmp/dsim.%s-w.log", id), "The worker id")

	var lookahead float64
	flag.Float64Var(&lookahead, "lookahead", 1, "The lookahead")

	flag.Parse()

	if resultPath == "" {
		log.Fatalf("resultpath argument is mandatory")
	}

	nodeConfig := dsim.SimulationNodeConfig{
		ListenAddress: listenAddress,
		ClockLogConfig: clock.ClockLogConfig{
			Priority:    clock.DEBUG,
			FileOutput:  true,
			LogFilename: logfile,
		},
		SimulationEngineConfig: dsim.SimulationEngineConfig{
			Lookahead:  dsim.Clock(lookahead),
			ResultPath: resultPath,
		},
	}

	log.Printf("Received listenAddress: %s\n", listenAddress)

	// Notify the sigCh channel for SIGINT (Ctrl+C) and SIGTERM (termination) signals.
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	node := dsim.NewSimulationNode(id, nodeConfig)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// Goroutine to catch shutdown signals
	go func() {
		select {
		case <-sigCh:
			fmt.Println("main: interrupt received. cancelling context.")
		case <-node.Done():
			fmt.Printf("main: node done")
			os.Exit(0)
		}
		cancel()
	}()

	if _, err := node.Start(ctx); err != nil {
		cancel()
	}
	<-ctx.Done()
	if ctx.Err() != nil {
		log.Printf("Exited: %v", ctx.Err())
		os.Exit(1)
	}
}
