package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
)

var tmpl = template.Must(template.New("index").Parse(`
<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Comandas App | HTMX + Go</title>
    <script src="https://unpkg.com/htmx.org@1.9.11"></script>
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        body { background-color: #09090b; color: #fafafa; }
    </style>
</head>
<body class="flex min-h-screen flex-col items-center justify-center p-6 text-center">
    <div class="relative flex flex-col items-center gap-6">
        <div class="absolute -inset-10 bg-blue-500/20 blur-3xl rounded-full"></div>
        
        <h1 class="text-6xl font-bold tracking-tighter sm:text-7xl bg-gradient-to-b from-white to-zinc-500 bg-clip-text text-transparent">
            Comandas App
        </h1>
        
        <p class="max-w-md text-zinc-400 text-lg sm:text-xl">
            Lógica en <span class="text-blue-400">Go</span>, interactividad con <span class="text-blue-400">HTMX</span>. 
            Sin carpetas de frontend, sin complicaciones.
        </p>

        <div id="status" class="mt-4 p-4 border border-zinc-800 rounded-xl min-w-[200px]">
            Haga clic para probar la conexión con Go
        </div>

        <div class="flex gap-4 mt-4">
            <button 
                hx-get="/api/test" 
                hx-target="#status" 
                class="px-8 py-3 bg-white text-black font-semibold rounded-full hover:bg-zinc-200 transition-colors">
                Probar API
            </button>
            <button class="px-8 py-3 border border-zinc-800 rounded-full hover:bg-zinc-900 transition-colors">
                Configuración
            </button>
        </div>
    </div>
</body>
</html>
`))

func main() {
	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@db:5432/comandadb"
	}

	// For simplicity in this example, we log the connection status
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
	} else {
		defer conn.Close(context.Background())
		fmt.Println("Connected to PostgreSQL successfully")
	}

	// Routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, nil)
	})

	http.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("¡Conexión Exitosa con Go! 🚀"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
