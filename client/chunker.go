package client

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
)

// printProgressBar prints a visual progress bar to the terminal using carriage return (\r).
func printProgressBar(bytesProcessed int64, totalBytes int64) {
	if totalBytes <= 0 {
		return
	}
	// Calculate percentage
	pct := (float64(bytesProcessed) / float64(totalBytes)) * 100
	if pct > 100 {
		pct = 100
	}
	// Bar configuration (20 characters wide)
	barWidth := 20
	filled := int(pct / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	// Construct the bar string
	// \u2588 is the Unicode code for the solid block █
	bar := strings.Repeat("█", filled) + strings.Repeat("-", barWidth-filled)
	// Print with \r at the beginning to overwrite the current line
	// Note: We use Printf and NO newline (\n) at the end.
	fmt.Printf("\r[%s] %.1f%% (%d/%d bytes)", bar, pct, bytesProcessed, totalBytes)
}

// ComputeFileHash calculates the SHA-256 has of a local file
func ComputeFileHash(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	buffer := make([]byte, 4096)

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			hasher.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}

		if err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// SendFileStream reads a file from disk and streams its bytes into the network connection.
func SendFileStream(filePath string, writer io.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Failed to open file for streaming: %v", err)

	}
	defer file.Close()

	//Get this file for progress calculation
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stats: %v", err)
	}
	totalSize := fileInfo.Size()

	buffer := make([]byte, 4096)
	var bytesSent int64

	// Print initial 0% progress bar
	printProgressBar(0, totalSize)

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			//Write the chunk directly to the network connection (writer)
			_, writeErr := writer.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write chunk to network: %v", writeErr)
			}
			bytesSent += int64(n)
			// Print updated progress bar
			printProgressBar(bytesSent, totalSize)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read file: %v", err)
		}
	}

	// Print a final newline so the next console outputs start on a new line!
	fmt.Println()
	return nil
}

//RecieverFileStream reads bytes from the network connection (reader) and writes them to the disk, while calculating the SHA-256 hash on the fly

func RecieverFileStream(dstPath string, reader io.Reader, fileSize int64) (string, error) {
	file, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %v", err)
	}
	defer file.Close()

	hasher := sha256.New()
	//use our favorite Y-splitter to write to the file and hasher simultaneously!
	multiWriter := io.MultiWriter(file, hasher)

	buffer := make([]byte, 4096)
	var bytesReceived int64

	for {
		//Read a chunk from the network connection
		n, err := reader.Read(buffer)
		if n > 0 {
			//Write the chunk to both files and hasher
			_, writeErr := multiWriter.Write(buffer[:n])
			if writeErr != nil {
				return "", fmt.Errorf("failed to write chunk to disk/hasher: %v", writeErr)
			}
			bytesReceived += int64(n)
			// Print updated progress bar
			printProgressBar(bytesReceived, fileSize)
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return "", fmt.Errorf("network error during read: %v", err)
		}
	}
	// Return the final hash
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// CopyFileAndHash reads a source file in 4KB chunks, writes them to a destination file, and calculates the SHA-256 hash of the content on the fly. It returns the hexadecimal represenation of the calculated SHA-256 hash.
func CopyFileAndHash(srcPath string, dstPath string) (string, error) {
	//open the source file for reading
	//os.Open returns an *os.file pointer and an error
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("Failed to Open sorce file: %v", err)
	}

	//defer tells Go to run this function right before the surrounding function returns, this ensures we always close the file and free up system resources.
	defer srcFile.Close()

	//Create the destination file for writing
	dstFile, err := os.Create(dstPath)
	if err != nil {
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
