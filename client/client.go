package client

import (
	"fmt"
	"path/filepath"
)

// ShareFile will handle the logic for slicing the file and hosting/sending it.
// We capitalize "ShareFile" so it is public (exported) and can be used in main.go.
func ShareFile(filePath string) {
	fmt.Printf("Client: Preparing to share file: %s\n", filePath)

	//For our local prototype, we will 	copy the file to a "downloaded_" version in the same directory to verify it works. 
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	dstPath := filepath.Join(dir, "copy_"+base)

	fmt.Printf("Prototype: copying file to %s to test chunking/hashing...\n", dstPath)
	hash, err := CopyFileAndHash(filePath, dstPath)
	if err !=nil {
			fmt.Printf("Error during prototype transfer: %v\n", err)
		return
	}
		fmt.Printf("Verification Successful!\n")
	fmt.Printf("File SHA-256 Fingerprint: %s\n", hash) 
}


// DownloadFile will handle connecting to the server and saving the incoming chunks.
func DownloadFile(code string) {
	fmt.Printf("Client: Attempting to download file with code: %s\n", code)
}