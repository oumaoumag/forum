package main

import (
	"log"
	"net/http"
)

func main() {
	// Create a new ServeMux to handle routing
	mux := http.NewServeMux()

	// Create a file server to serve static files from the "web/static" directory
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	server := &http.Server{
		Addr:    ":8080", // Listen on port 8080
		Handler: mux,     // Use the defined ServeMux as the handler
	}

	log.Println("Server started at http://localhost:8080")
	// Start the server and log any errors
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
