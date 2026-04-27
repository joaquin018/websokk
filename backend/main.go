package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@db:5432/comandadb"
	}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
	} else {
		defer conn.Close(context.Background())
		fmt.Println("Connected to PostgreSQL successfully")
	}

	// Routes
	// 'handleIndex' vuelve a estar disponible localmente
	http.HandleFunc("/", handleIndex)

	// API Endpoints
	http.HandleFunc("/api/test", handleAPITest)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleAPITest(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("¡Conexión Exitosa con Go! 🚀"))
}
