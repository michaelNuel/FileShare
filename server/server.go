package server

import (
	"fmt"
)

// Start kicks off our signaling and relay server.

func Start(port string) {
	fmt.Printf("Server: Starting signing & relay server on port %s....\n", port) 
}