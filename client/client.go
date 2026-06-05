package client

import (
	"fmt"
)

// ShareFile will handle the logic for slicing the file and hosting/sending it.
// We capitalize "ShareFile" so it is public (exported) and can be used in main.go.
func ShareFile(filePath string) {
	fmt.Printf("Client: Preparing to share file: %s\n", filePath)
}


// DownloadFile will handle connecting to the server and saving the incoming chunks.
func DownloadFile(code string) {
	fmt.Printf("Client: Attempting to download file with code: %s\n", code)
}