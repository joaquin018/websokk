package main

import (
	"context"
	"log"
	"os"
	"time"

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

// Inicializar Base de Datos con Reintento
func initDB() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@db:5432/comandadb"
	}

	log.Printf("Intentando conectar a: %s\n", dbURL)

	var err error
	// Intentar conectar hasta 5 veces (por si la DB tarda en arrancar)
	for i := 0; i < 5; i++ {
		db, err = pgx.Connect(context.Background(), dbURL)
		if err == nil {
			break
		}
		log.Printf("Intento %d fallido, reintentando en 2s...\n", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Printf("❌ ERROR FINAL: No se pudo conectar a la DB: %v\n", err)
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
