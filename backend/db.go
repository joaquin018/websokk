package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

// --- MODELOS ---
type Product struct {
	ID          int
	Name        string
	Price       int
	Description string
}

type Order struct {
	ID     int
	Plato  string
	Mesa   string
	Estado string
}

// Global DB Connection
var db *pgx.Conn

// Inicializar Base de Datos
func initDB() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@db:5432/comandadb"
	}

	var err error
	db, err = pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Printf("❌ Error conectando a la DB: %v\n", err)
		return
	}

	// Crear tablas si no existen
	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY, 
			plato TEXT, 
			mesa TEXT, 
			estado TEXT DEFAULT 'pendiente'
		);
		CREATE TABLE IF NOT EXISTS products (
			id SERIAL PRIMARY KEY, 
			name TEXT NOT NULL, 
			price INTEGER NOT NULL, 
			description TEXT
		);
	`)
	if err != nil {
		log.Printf("❌ Error creando tablas: %v\n", err)
	} else {
		log.Println("✅ Base de datos conectada y tablas listas")
	}
}
