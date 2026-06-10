package client

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ZipFolder zips a folder recursively into a single target .zip file
func ZipFolder(sourceFolder string, targetZipPath string) error {
	//Create the destination zip file
	zipFile, err := os.Create(targetZipPath)
	if err != nil {
		return fmt.Errorf("Failed to create zip archive %v", err)
	}
	defer zipFile.Close()

	//Create a Zip Writer that compresses bytes as they are written
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close() //defer close seals the zip file at the end

	//Walk the directory tree recursively
	err = filepath.Walk(sourceFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		//skip  directories (we only write files; folders will be recreated during unzip)
		if info.IsDir() {
			return nil
		}
		//get the relative path of the file (/) for ZIP specification compatability
		relPath, err := filepath.Rel(sourceFolder, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %v", err)
		}

		// Convert paths to use forward slashes (/) for ZIP specification compatibility

		zipPath := filepath.ToSlash(relPath)
		// Create a file header inside the zip archive
		writerInZip, err := zipWriter.Create(zipPath)
		if err != nil {
			return fmt.Errorf("Failed to create file entry in zip: %v", err)
		}

		//Open the actual source file on disk
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file for zipping: %v", err)
		}
		defer file.Close()

		// Stream the file bytes directly into the zip entry
		_, err = io.Copy(writerInZip, file)
		if err != nil {
			return fmt.Errorf("Failed to write file content to zip: %v", err)
		}

		return nil

	})

	if err != nil {
		return fmt.Errorf("directory walk failed %v", err)
	}

	return nil
}

func UnZipFolder(zipFilePath string, destinationDir string) error {
	//open the zip file reader
	reader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return fmt.Errorf("failed to open zip file for reading: %v", err)
	}
	defer reader.Close()

	//loop through all files in the zip archive

	for _, file := range reader.File {
		//Calculate the final file path on disk
		extractedFilePath := filepath.Join(destinationDir, file.Name)

		// Safety Check: Prevent Zip Slip security vulnerability.
		// Ensure the path does not try to traverse outside the destination directory.

		cleanDestDir := filepath.Clean(destinationDir)
		if !strings.HasPrefix(extractedFilePath, cleanDestDir) {
			return fmt.Errorf("security error: path traversal detected: %s", file.Name)
		}

		// Check if it's a directory entry inside the zip (some zipping tools create these)
		if file.FileInfo().IsDir() {
			err := os.MkdirAll(extractedFilePath, file.Mode())
			if err != nil {
				return fmt.Errorf("failed to create directory: %v", err)
			}

			continue
		}

		//Create parent directories if they dont exist yet
		parentDir := filepath.Dir(extractedFilePath)
		err = os.Mkdir(parentDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create parent directories: %v", err)
		}

		//Open the file inside the zip archive
		fileReader, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open zipped file: %v", err)
		}

		//Create the empty target file on the local hard drive, preserving original file permisions
		dstFile, err := os.OpenFile(extractedFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			fileReader.Close()
			return fmt.Errorf("failed to create output file: %v", err)
		}

		//Copy the uncompressed data into the new file on disk
		_, err = io.Copy(dstFile, fileReader)
		dstFile.Close()
		fileReader.Close()
		if err != nil {
			return fmt.Errorf("failed to decompress file: %v", err)
		}
	}

	return nil
}

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
