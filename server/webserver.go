package server

import (
	"fmt"
	"net/http"
)

//StartWeb starts the HTTP server to serve the frontend and handle WebSockets. 

func StartWeb(port string) {
	//Create a file server handler pointer to our "./web" directory
	fileServer := http.FileServer(http.Dir("./web")) 
    

	//Map the root path "/" to our file server 
	http.Handle ("/", fileServer)

	fmt.Printf("Server: Web Ui is live at http://localhost:%s\n", port)

	//Start the Http server 
	//If it fails, listenAndServe returns an error, which we print. 
	err := http.ListenAndServe(":"+port,  fileServer)
	if err != nil {
		fmt.Printf("Web Server Error: %v\n", err)
	}
	
}