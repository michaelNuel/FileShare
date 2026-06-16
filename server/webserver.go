package server

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Websession tracks a WebSocket-based file sharing operation
type WebSession struct {
	fileName      string
	fileSize      int64
	fileHash      string
	senderConn    *websocket.Conn
	receiverChan  chan *websocket.Conn //passes the receiver's websocket to the sender's goroutine
}

// Global state to store WebSocket sessions
var (
	WebSessions  = make(map[string]*WebSession)
	webSessionMu sync.Mutex
)

// Create a webSocket upgrader.Allow all origins so external devices can connect
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true //Allow cross-origin requests
	},
}

//StartWeb starts the HTTP server to serve the frontend and handle WebSockets.

func StartWeb(port string) {
	//Create a file server handler pointer to our "./web" directory
	fileServer := http.FileServer(http.Dir("./web"))

	//Map the root path "/" to our file server
	http.Handle("/", fileServer)

	// Map our WebSocket endpoint
	http.HandleFunc("/ws", handleWebSocket)

	fmt.Printf("Server: Web Ui is live at http://localhost:%s\n", port)
	fmt.Printf("Server: WebSocket endpoint is active at ws://localhost:%s/ws\n", port)

	//Start the Http server
	//If it fails, listenAndServe returns an error, which we print.
	err := http.ListenAndServe(":"+port, fileServer)
	if err != nil {
		fmt.Printf("Web Server Error: %v\n", err)
	}

}

//handleWebSocket handles WebSocket handshake and routing.

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	//Upgarde the Http  connection to a WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("WebSocket Upgrade Error: %v\n", err)
		return
	}

	defer ws.Close()

	//Read the first command message (expected to be TEXT)
	msgType, payload, err := ws.ReadMessage()
	if err != nil {
		return
	}

	if msgType != websocket.TextMessage {
		ws.WriteMessage(websocket.TextMessage, []byte("Error expected text command"))
		return
	}

	line := strings.TrimSpace(string(payload))
	parts := strings.Split(line, " ")
	if len(parts) < 2 {
		ws.WriteMessage(websocket.TextMessage, []byte("ERROR invalid command structure"))
		return
	}

	command := parts[0]

	switch command {
	case "SHARE":
		// Format: SHARE <filename> <filesize> <filehash>
		if len(parts) < 4 {
			ws.WriteMessage(websocket.TextMessage, []byte("ERROR usage: SHARE <filename> <filesize> <filehash>"))
			return
		}
		handleWebShare(ws, parts[1], parts[2], parts[3])
	case "DOWNLOAD":
		// Format: DOWNLOAD <code>
		handleWebDownload(ws, parts[1])
	default:
		ws.WriteMessage(websocket.TextMessage, []byte("ERROR unknown command"))
	}

}


//handleWebShare  manages the sender's Websocket session 
func handleWebShare(senderWs *websocket.Conn, fileName string, sizeStr string, fileHash string) {
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		senderWs.WriteMessage(websocket.TextMessage, []byte("Error invalid file size"))
		return 
	}

	//Generate code and create session 
	code := generateCode()
	session := &WebSession{
		fileName:     fileName,
		fileSize:     size,
		fileHash:     fileHash,
		senderConn:   senderWs,
		receiverChan: make(chan *websocket.Conn, 1),
	}


	//Save to map 
	//Store the new Session
	webSessionMu.Lock()
	WebSessions[code] = session
	webSessionMu.Unlock()

		fmt.Printf("Web Server: Code %s generated for %s (%d bytes)\n", code, fileName, size)

			// Send code back to sender
	senderWs.WriteMessage(websocket.TextMessage, []byte("CODE "+code))


		// Block and wait for receiver
	select {
	case receiverWs := <-session.receiverChan:
		// Tell sender to start streaming binary blocks
		senderWs.WriteMessage(websocket.TextMessage, []byte("START"))
		fmt.Printf("Web Server: Transfer starting for code %s\n", code)
		// Relay loop: Read binary chunks from sender and write them to receiver
		for {
			t, data, err := senderWs.ReadMessage()
			if err != nil {
				// Sender disconnected or completed transfer
				break
			}
			// Forward the frame directly to receiver
			err = receiverWs.WriteMessage(t, data)
			if err != nil {
				break
			}
		}
		fmt.Printf("Web Server: Transfer ended for code %s\n", code)
	case <-time.After(5 * time.Minute):
		fmt.Printf("Web Server: Session %s timed out\n", code)
		senderWs.WriteMessage(websocket.TextMessage, []byte("ERROR timeout"))
	}
	// Clean up session
	webSessionMu.Lock()
	delete(WebSessions, code)
	webSessionMu.Unlock()
}

// handleWebDownload manages the receiver's WebSocket session

func handleWebDownload(receiverWs *websocket.Conn, code string) { 
	// Look up session
	webSessionMu.Lock()
	session, exists := WebSessions[code]
	webSessionMu.Unlock()
	if !exists {
		receiverWs.WriteMessage(websocket.TextMessage, []byte("ERROR session not found"))
		return
	}
	// Send metadata to receiver
	metadataMsg := fmt.Sprintf("METADATA %s %d %s", session.fileName, session.fileSize, session.fileHash)
	err := receiverWs.WriteMessage(websocket.TextMessage, []byte(metadataMsg))
	if err != nil {
		return
	}
	// Wait for "READY" text message
	t, payload, err := receiverWs.ReadMessage()
	if err != nil || t != websocket.TextMessage || strings.TrimSpace(string(payload)) != "READY" {
		return
	}
	// Wake up sender
	session.receiverChan <- receiverWs
	// Keep connection alive while sender relays. We block here until connection is closed.
	for {
		_, _, err := receiverWs.ReadMessage()
		if err != nil {
			break
		}
	}
}
