package server

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TransferSession struct {
	fileName     string
	fileSize     int64
	fileHash     string
	senderConn   net.Conn
	receiverChan chan net.Conn // Used to pass the receiver's connection to the sender's goroutine
}

// Global state to store the sessions . Maps code -> Session.
var (
	sessions  = make(map[string]*TransferSession)
	sessionsMu sync.Mutex //lock to ensure only one client updates the sessions map at a time, thereby protecting the sessions map from concurrent access crashes

)

// Start starts the TCP signaling and relay server.

func Start(port string) {
	//Listen on all interfaces on the specified TCP port
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Printf("Sever Error: Failed to start server: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Printf("Server: Listener on port %s...\n", port)

	//Loop forever, accepting clients connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Server Error: Failed to accept connection: %v\n", err)
			continue
		}

		//Handle each connection concurrently in a background thread (goroutine)
		go handleConnection(conn)

	}

}

func handleConnection(conn net.Conn) {
	//Ensure we close the connection if we exit early
	//Note: If the connection is handed over to the relay, the relay will handle closing

	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	//Read the first line of text from the connection
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	//Trim the newline Characters and split by spaces
	line = strings.TrimSpace(line)
	parts := strings.Split(line, " ")
	if len(parts) < 2 {
		fmt.Fprintln(conn, "Error invalid command structure")
		return
	}

	command := parts[0]

	switch command {
	case "SHARE":
		//Format: Share <filename> <filesize> <filehash>
		if len(parts) < 4 {
			fmt.Fprintln(conn, "Error usage: Share <filename> <filesize> <filehash>")
			return
		}
		handleShare(conn, parts[1], parts[2], parts[3])
		conn = nil //set to nil so defer doesnt close it; handle share will handle closing

	case "DOWNLOAD":
		//Format: Download <code>

		handleDownload(conn, parts[1])
		conn = nil // Set to nil so defer doesn't close it; handleDownload will handle close

	default:
		fmt.Fprintln(conn, "Error unknown command")

	}
}

func handleShare(senderConn net.Conn, fileName string, sizeStr string, fileHash string) {
	defer senderConn.Close()

	//Convert file size from string to integer
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		fmt.Fprintln(senderConn, "Error invalid file size")
		return
	}
	//Generate a 6-digit code
	code := generateCode()

	//create a new session
	session := &TransferSession{
		fileName:     fileName,
		fileSize:     size,
		fileHash:     fileHash,
		senderConn:   senderConn,
		receiverChan: make(chan net.Conn, 1), //Buffered channel size 1
	}

	//save to global map (protected by Mutex lock)
	sessionsMu.Lock()
	sessions[code] = session
	sessionsMu.Unlock()

	fmt.Printf("Server: Code %s generated for file %s (%d bytes)\n", code, fileName, size)

	//send the code back to the sender
	fmt.Fprintf(senderConn, "CODE %s\n", code)

	//Block here and wait for a reciever to join
	//This will wait until handleDownload 	writes to session.recieverChan.
	//We set a 5-minute timeout so sessions dont hang forever if no one downloads
	select {
	case receiverConn := <-session.receiverChan:
		defer receiverConn.Close()

		//Tell the server to start sending raw file bytes
		fmt.Fprintln(senderConn, "START")
		fmt.Printf("Server: Transfer started for code %s\n", code)

		//relay bytes from sender to reciever
		bytesCopied, err := io.Copy(receiverConn, senderConn)
		if err != nil {
			fmt.Printf("Server Error: Transfer failed for code %s: %v\n", code, err)
		} else {
			fmt.Printf("Server: Transfer completed for code %s (%d bytes relayed)\n", code, bytesCopied)
		}

	case <-time.After(5 * time.Minute):
		fmt.Printf("Server: Session %s timed out waiting for receiver\n", code)
		fmt.Fprintln(senderConn, "ERROR timeout waiting for receiver")
	}

	//Clean up map
	sessionsMu.Lock()
	delete(sessions, code)
	sessionsMu.Unlock()

}

func handleDownload(receiverConn net.Conn, code string) {
	//Look up session (protected by Lock)
	sessionsMu.Lock()
	session, exists := sessions[code]
	sessionsMu.Unlock()

	if !exists {
		fmt.Fprintln(receiverConn, "Error session not found")
		receiverConn.Close()
		return
	}

	//send metadata to reciever
	fmt.Fprintf(receiverConn, "METADATA %s %d %s\n", session.fileName, session.fileSize, session.fileHash)

	//Read "READY\n" from receiver
	reader := bufio.NewReader(receiverConn)
	line, err := reader.ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "READY" {
		receiverConn.Close()
		return
	}

	//send the reciever's connection to the sender's goroutine.
	//This wakes up the select block in the handleShare !
	session.receiverChan <- receiverConn
}

// generateCode generates a temporary 6-digit random code
func generateCode() string {
	rand.Seed(time.Now().UnixNano())
	code := rand.Intn(900000) + 100000 //100000 to 999999 (I have questions here )
	return fmt.Sprintf("%d", code)
}
