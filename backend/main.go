package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
)

// WebSocket Hub
type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				_ = client.WriteMessage(websocket.TextMessage, message)
			}
			h.mu.Unlock()
		}
	}
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@db:5432/comandadb"
	}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Printf("DB Error: %v\n", err)
	} else {
		defer conn.Close(context.Background())
		_, _ = conn.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS orders (id SERIAL PRIMARY KEY, plato TEXT, mesa TEXT, estado TEXT);`)
	}

	hub := newHub()
	go hub.run()

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/comandas", handleComandas)
	http.HandleFunc("/cocina", handleCocina)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		c, _ := upgrader.Upgrade(w, r, nil)
		hub.register <- c
		go func() {
			defer func() { hub.unregister <- c }()
			for {
				if _, _, err := c.ReadMessage(); err != nil {
					break
				}
			}
		}()
	})

	// Crear Pedido
	http.HandleFunc("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			plato, mesa := r.FormValue("plato"), r.FormValue("mesa")
			var id int
			_ = conn.QueryRow(context.Background(), "INSERT INTO orders (plato, mesa, estado) VALUES ($1, $2, 'pendiente') RETURNING id", plato, mesa).Scan(&id)

			// Enviar a todos via WS
			html := fmt.Sprintf(`<div id="pedidos-col" hx-swap-oob="beforeend">%s</div>`, RenderOrderCard(id, mesa, plato, "pendiente"))
			hub.broadcast <- []byte(html)

			// Reset formulario
			w.Write([]byte(`
				<form hx-post="/api/orders" hx-swap="none" hx-on::after-request="this.reset()" class="flex flex-col gap-4">
					<input type="text" name="mesa" placeholder="Mesa" class="bg-zinc-900 border border-zinc-800 p-4 rounded-2xl outline-none" required>
					<textarea name="plato" placeholder="Pedido" class="bg-zinc-900 border border-zinc-800 p-4 rounded-2xl outline-none h-32" required></textarea>
					<button type="submit" class="bg-blue-600 py-4 rounded-2xl font-bold transition-all mt-2">Enviar 🚀</button>
				</form>`))
		}
	})

	// Actualizar Pedido
	http.HandleFunc("/api/orders/update/", func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/api/orders/update/")
		id, _ := strconv.Atoi(idStr)
		newStatus := r.URL.Query().Get("status")

		if newStatus == "delete" {
			_, _ = conn.Exec(context.Background(), "DELETE FROM orders WHERE id=$1", id)
			hub.broadcast <- []byte(fmt.Sprintf(`<div id="order-%d" hx-swap-oob="delete"></div>`, id))
			return
		}

		var mesa, plato string
		_ = conn.QueryRow(context.Background(), "UPDATE orders SET estado=$1 WHERE id=$2 RETURNING mesa, plato", newStatus, id).Scan(&mesa, &plato)

		// Mover tarjeta via WS (OOB swap)
		targetCol := "proceso-col"
		if newStatus == "completado" {
			targetCol = "completado-col"
		}

		// Borramos de la anterior y añadimos a la nueva
		html := fmt.Sprintf(`
			<div id="order-%d" hx-swap-oob="delete"></div>
			<div id="%s" hx-swap-oob="beforeend">%s</div>`, id, targetCol, RenderOrderCard(id, mesa, plato, newStatus))
		
		hub.broadcast <- []byte(html)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server on %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
