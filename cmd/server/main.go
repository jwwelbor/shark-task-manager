package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/jwwelbor/shark-task-manager/internal/db"
)

func main() {
	// Initialize database
	database, err := db.InitDB("shark-tasks.db")
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	log.Println("Database initialized successfully")

	// Run integrity check
	if err := db.CheckIntegrity(database); err != nil {
		log.Fatal("Database integrity check failed:", err)
	}
	log.Println("Database integrity check passed")

	// Set up routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Shark Task Manager API - Database Ready")
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Check database connection
		if err := database.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Database unavailable: %v", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Start server
	port := "8080"
	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
