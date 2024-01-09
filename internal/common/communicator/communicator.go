package communicator

import (
	"encoding/gob"
	"fmt"
	"net"
)

type ConnectionHandlerCallback func(net.Conn)

func HandleConnections(listener net.Listener, handler ConnectionHandlerCallback) {
	// Main loop to handle connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			// Check if the error is due to listener closure
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				return // Listener was closed
			}
			return
		}
		go handler(conn)
	}
}

func Send(conn net.Conn, message interface{}) error {
	encoder := gob.NewEncoder(conn)
	return encoder.Encode(&message)
}

func Receive(conn net.Conn) (interface{}, error) {
	var data interface{}
	decoder := gob.NewDecoder(conn)
	err := decoder.Decode(&data)
	return data, err
}

func SendReceiveTCP(address string, message interface{}) (interface{}, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("error connecting to node: %v", err)
	}
	encoder := gob.NewEncoder(conn)
	if err = encoder.Encode(&message); err != nil {
		return nil, fmt.Errorf("error sending event to node: %v", err)
	}

	return Receive(conn)
}
