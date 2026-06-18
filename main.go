package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/michaelnuel/fileshare/client"
	"github.com/michaelnuel/fileshare/server"
)

func main() {
	//Define the Flag sets for each  command
	// flag.ExitOnError tells Go to stop the program and show help text if the user types an invalid flag.
	shareCmd := flag.NewFlagSet("share", flag.ExitOnError)
	downloadCmd := flag.NewFlagSet("download", flag.ExitOnError)
	serverCmd := flag.NewFlagSet("server", flag.ExitOnError)

	//Define the specific variables to capture command line arguements
	var filePath string
	var code string
	// var port string
	var serverAddr string
	var runWeb bool
		// Check if the cloud provider injected a PORT environment variable.
	// If it exists, use it. Otherwise, default to "8080".
		port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	//Bind the Variable to the flag set
	// The "&" symbol is a pointer, telling Go where to save the values in memory.
	shareCmd.StringVar(&filePath, "file", "", "Path to the file or folder to share (Required)")
	shareCmd.StringVar(&serverAddr, "server", "localhost:8080", "Relay server address (IP:port or domain:port)")
	downloadCmd.StringVar(&code, "code", "", "6-digit download code (Required)")
	downloadCmd.StringVar(&serverAddr, "server", "localhost:8080", "Relay server address (IP:port or domain:port)")
	serverCmd.StringVar(&port, "port", "8080", "Port to run the signaling server on")
	serverCmd.BoolVar(&runWeb, "web", false, "Run is Web mode (starts HTTP/WebSocket server)") 
	//Makes sure a user enters a command
	if len(os.Args) < 2 {
		printGeneralUsage()
		os.Exit(1)
	}

	//Look at the first argument to determine which subcommand to parse
	switch os.Args[1] {
	case "share":
		//Parse parses flag definitions from the argument list (excluding the command name itself)
		shareCmd.Parse(os.Args[2:])
		if filePath == "" {
			fmt.Println("Error: You must Provide a file path to share. ")
			shareCmd.PrintDefaults() //	Prints the list of the flags and description
			os.Exit(1)
		}

		//Call our client package's function!
		client.ShareFile(filePath, serverAddr)

	case "download":
		downloadCmd.Parse(os.Args[2:])
		if code == "" {
			fmt.Println("Error: you must provide a 6 digit code.")
			downloadCmd.PrintDefaults()
			os.Exit(1)  
		}
		client.DownloadFile(code, serverAddr)

	case "server":
		serverCmd.Parse(os.Args[2:])
		if runWeb {
			server.StartWeb(port)// Start our new Web server!
		} else {
			server.Start(port)  //Start the Original TCP server!
		} 
	
 
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printGeneralUsage()
		os.Exit(1)
	}
}

// A helper function to print the general commands available.
func printGeneralUsage() {
	fmt.Println("Usage: fileshare <command> [<args>]")
	fmt.Println("Commands:")
	fmt.Println("  share     - Share a file/folder (Requires: -file <path>, Optional: -server <addr>)")
	fmt.Println("  download  - Download a file/folder (Requires: -code <code>, Optional: -server <addr>)")
	fmt.Println("  server    - Start the signaling/relay server (Optional: -port <port>)")
}
