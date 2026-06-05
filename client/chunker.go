package client

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)
// CopyFileAndHash reads a source file in 4KB chunks, writes them to a destination file, and calculates the SHA-256 hash of the content on the fly. It returns the hexadecimal represenation of the calculated SHA-256 hash. 
func CopyFileAndHash(srcPath string, dstPath string) (string, error) {
	//open the source file for reading 
	//os.Open returns an *os.file pointer and an error 
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("Failed to Open sorce file: %v", 	err)		
	}

	//defer tells Go to run this function right before the surrounding function returns, this ensures we always close the file and free up system resources. 
	defer srcFile.Close()

	//Create the destination file for writing 
	dstFile, err := os.Create(dstPath)
	if err != nil{
		return "", fmt.Errorf("Failed to create destination file: %v", err)
	}

	defer dstFile.Close()

	//Create a SHA-256 hasher
	hasher := sha256.New()

	//Combine the destination file and the hasher using a MultiWriter 
	//Any write to multiWriter will write to both dstFile and hasher 

	multiWriter := io.MultiWriter(dstFile, hasher)

	//Create our 4kb byte buffer 
	//make() is used to allocate slices, maps, and channels 
	//byte is an alias for uint8 (an 8-bit unsigned integer). 
	buffer := make([]byte, 4096)

	var totalBytesRead int64

	//Start the chunking loop

	for {
		//Read up to 4096 bytes from the source file into our buffer. 
		//n is the actual number of bytes read (could be less than 4096 at the end of the file).
		n, readErr := srcFile.Read(buffer)
		if n > 0 {
			//Write Exactly the n bytes we read (buffer[:n]) to our multiWriter. 
			//IF we wrote the whole buffer, we might write leftover garbage bytes!
			_, writeErr := multiWriter.Write(buffer[:n])
			if writeErr != nil {
				return "", fmt.Errorf("failed to write data: %v", writeErr)
			}
			totalBytesRead += int64(n) 
		}

		//Check if we hit the End of file (EOF)
		if readErr == io.EOF {
			break //Exit the loop safely 
		}

		if readErr != nil {
			return "", fmt.Errorf("Failed to read data: %v", readErr)
		}
	}

		// Get the final SHA-256 hash bytes.
	// Sum(nil) appends the current hash to the passed slice (nil means we just want the hash itself).
	hashBytes := hasher.Sum(nil)

	//Convert the bytes to a hexadecimal string. 
	// %x formats bytes as a hex string (e.g "a3f5b2..."). 
	hashString := fmt.Sprintf("%x", hashBytes)

	fmt.Printf("Prototype: Successfully copied %d bytes. \n", totalBytesRead)
	return hashString, nil 
}