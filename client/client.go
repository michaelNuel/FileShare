package client

import (
	"bufio"
	"fmt"	
	"net"
	"os"
	"path/filepath"
	"strings"
)

// ShareFile will handle the logic for slicing the file and hosting/sending it.
// We capitalize "ShareFile" so it is public (exported) and can be used in main.go.
func ShareFile(filePath string) {
	fmt.Printf("Client: Preparing to share file: %s\n", filePath)
    //Get the file information (name and size)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("Error: Failed to get file details: %v\n", err)
		return
	}
	fileName := filepath.Base(filePath)
	fileSize :=fileInfo.Size()

	//Compute the file's SHA-256 fingerprint
	fmt.Println("Client: Calculating file hash fingerprint...")
	hash, err := ComputeFileHash(filePath)
	if err != nil {
		fmt.Printf("Error: Failed to calculate file hash: %v\n", err)
		return
	}
	
	//Connect to the signalling & relay server 
	conn, err :=net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Printf("Error: failed to connect to server: %v\n", err)
		return 
	}
	defer conn.Close()

	//send the share command to register the file 
	fmt.Fprintf(conn, "Share %s %d %s\n", fileName, fileSize, hash)

  
}


// DownloadFile will handle connecting to the server and saving the incoming chunks.
func DownloadFile(code string) {
	fmt.Printf("Client: Attempting to download file with code: %s\n", code)
}