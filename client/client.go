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
	fileSize := fileInfo.Size()

	//Compute the file's SHA-256 fingerprint
	fmt.Println("Client: Calculating file hash fingerprint...")
	hash, err := ComputeFileHash(filePath)
	if err != nil {
		fmt.Printf("Error: Failed to calculate file hash: %v\n", err)
		return
	}

	//Connect to the signalling & relay server
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Printf("Error: failed to connect to server: %v\n", err)
		return
	}
	defer conn.Close()

	//send the share command to register the file
	fmt.Fprintf(conn, "SHARE %s %d %s\n", fileName, fileSize, hash)

	//Read the 6-digit code response from the server
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error: Failed to read server response: %v\n", err)
		return
	}

	response = strings.TrimSpace(response)
	parts := strings.Split(response, " ")

	if parts[0] != "CODE" || len(parts) < 2 {
		fmt.Printf("Server Error: %s\n", response)
		return
	}

	code := parts[1]
	fmt.Println("------------------------------------")
	fmt.Printf("SUCCESS! File is ready to share.\n")
	fmt.Printf("Give this 6-digit code to your friend: %s\n", code)
	fmt.Println("Waiting for receiver to connect...")
	fmt.Println("--------------------------------------------------")

	//Block and wait for the Start command from the server
	status, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error: Connection lost: %v\n", err)
		return
	}

	status = strings.TrimSpace(status)
	if status == "START" {
		fmt.Println("Receiver connected! Streaming file chunks...")

		//Stream the file  bytes into the network socket
		err = SendFileStream(filePath, conn)
		if err != nil {
			fmt.Printf("Error: File transfer failed: %v\n", err)
			return
		}
		fmt.Println("Transfer complete!")
	} else {
		fmt.Printf("Error: Unexpected server status: %s\n", status)
	}

}

// DownloadFile will handle connecting to the server and saving the incoming chunks.
func DownloadFile(code string) {
	fmt.Printf("Client: Attempting to download file with code: %s\n", code)

	//Connect to the signaling & relay server
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Printf("Error: Failed to connect to server: %v\n", err)
		return
	}
	defer conn.Close()

	//Send the Download command with the 6-digit code
	fmt.Fprintf(conn, "DOWNLOAD %s\n", code)

	//Read metadata response from server
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error: Failed to read server response: %v\n", err)
		return
	}

	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "ERROR") {
		fmt.Printf("Server Error: %s\n", response)
		return
	}

	//Format: METADATA <filename> <filesize> <filehash>
	parts := strings.Split(response, " ")
	if parts[0] != "METADATA" || len(parts) < 4 {
		fmt.Printf("Error: Invalid server response: %s\n", response)
		return
	}

	fileName := parts[1]
	fileSizeStr := parts[2]
	targetHash := parts[3]

	fmt.Println("--------------------------------------------------")
	fmt.Printf("File Found: %s (%s bytes)\n", fileName, fileSizeStr)
	fmt.Println("--------------------------------------------------")

	//Create the local output path (prefixed with "download_")
	downloadPath := "downlaoded_" + fileName
	fmt.Printf("Saving file to: %s\n", downloadPath)

	//Tell the server we are ready to recieve the stream
	fmt.Fprintln(conn, "READY")
	fmt.Println("Downloading file chunks...")

	//Stream file bytes from the network socket to disk
	calculatedHash, err := RecieverFileStream(downloadPath, conn)
	if err != nil {
     fmt.Printf("Error: Download failed: %v\n", err)
	 //Clean up the incomplete file
	 os.Remove(downloadPath)
	 return
	}

	//Verify the SHA-256 fingerprint matches the original
	fmt.Println("Verifying integrity (SHA-256 Checksum)...")
		if calculatedHash == targetHash { 
		fmt.Println("--------------------------------------------------")
		fmt.Println("SUCCESS! File downloaded and verified.")
		fmt.Printf("SHA-256: %s\n", calculatedHash)
		fmt.Println("--------------------------------------------------")
	} else {
		fmt.Println("--------------------------------------------------")
		fmt.Printf("WARNING: Verification failed! File is corrupted.\n")
		fmt.Printf("Expected Hash:   %s\n", targetHash)
		fmt.Printf("Calculated Hash: %s\n", calculatedHash)
		fmt.Println("Deleting corrupted file...")
		fmt.Println("--------------------------------------------------")
		os.Remove(downloadPath)
	}
}
