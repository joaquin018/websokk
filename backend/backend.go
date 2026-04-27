package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/gorilla/websocket"
)

// --- HELPERS DE TEXTO ---
func toTitleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		if len(w) > 0 {
			r := []rune(w)
			r[0] = unicode.ToUpper(r[0])
			words[i] = string(r)
		}
	}
	return strings.Join(words, " ")
}

func toSentenceCase(s string) string {
	if len(s) == 0 { return s }
	s = strings.ToLower(s)
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// --- WEBSOCKET HUB ---
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
	// Inicializar Base de Datos (desde db.go)
	initDB()
	if db != nil {
		defer db.Close(context.Background())
	}

	hub := newHub()
	go hub.run()

	// --- RUTAS ---
	http.HandleFunc("/", handleComandas)
	http.HandleFunc("/cocina", handleCocina)
	http.HandleFunc("/config", handleConfig)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		c, _ := upgrader.Upgrade(w, r, nil)
		hub.register <- c
		go func() {
			defer func() { hub.unregister <- c }()
			for {
				if _, _, err := c.ReadMessage(); err != nil { break }
			}
		}()
	})

	// --- API PRODUCTOS ---
	http.HandleFunc("/api/products", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			name := toTitleCase(r.FormValue("name"))
			description := toSentenceCase(r.FormValue("description"))
			price, _ := strconv.Atoi(r.FormValue("price"))

			_, _ = db.Exec(context.Background(), 
				"INSERT INTO products (name, price, description) VALUES ($1, $2, $3)", 
				name, price, description)
		}

		rows, _ := db.Query(context.Background(), "SELECT id, name, price, description FROM products ORDER BY id DESC")
		var products []Product
		for rows.Next() {
			var p Product
			rows.Scan(&p.ID, &p.Name, &p.Price, &p.Description)
			products = append(products, p)
		}
		RenderProductList(w, products)
	})

	// --- API PEDIDOS ---
	http.HandleFunc("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			plato, mesa := r.FormValue("plato"), r.FormValue("mesa")
			var id int
			_ = db.QueryRow(context.Background(), "INSERT INTO orders (plato, mesa, estado) VALUES ($1, $2, 'pendiente') RETURNING id", plato, mesa).Scan(&id)
			
			html := fmt.Sprintf(`<div id="pedidos-col" hx-swap-oob="beforeend">%s</div>`, RenderOrderCard(id, mesa, plato, "pendiente"))
			hub.broadcast <- []byte(html)
			
			w.Write([]byte(`
				<form hx-post="/api/orders" hx-swap="none" hx-on::after-request="this.reset()" class="flex flex-col gap-4">
					<input type="text" name="mesa" placeholder="Mesa" class="bg-zinc-900/50 border border-zinc-800 p-4 rounded-2xl outline-none" required>
					<textarea name="plato" placeholder="Pedido" class="bg-zinc-900/50 border border-zinc-800 p-4 rounded-2xl outline-none h-32" required></textarea>
					<button type="submit" class="bg-white text-black py-4 rounded-2xl font-bold mt-2">Enviar 🚀</button>
				</form>`))
		}
	})

	http.HandleFunc("/api/orders/update/", func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/api/orders/update/")
		id, _ := strconv.Atoi(idStr)
		newStatus := r.URL.Query().Get("status")
		
		if newStatus == "delete" {
			_, _ = db.Exec(context.Background(), "DELETE FROM orders WHERE id=$1", id)
			hub.broadcast <- []byte(fmt.Sprintf(`<div id="order-%d" hx-swap-oob="delete"></div>`, id))
			return
		}
		
		var mesa, plato string
		_ = db.QueryRow(context.Background(), "UPDATE orders SET estado=$1 WHERE id=$2 RETURNING mesa, plato", newStatus, id).Scan(&mesa, &plato)
		
		targetCol := "proceso-col"
		if newStatus == "completado" { targetCol = "completado-col" }
		
		html := fmt.Sprintf(`<div id="order-%d" hx-swap-oob="delete"></div><div id="%s" hx-swap-oob="beforeend">%s</div>`, id, targetCol, RenderOrderCard(id, mesa, plato, newStatus))
		hub.broadcast <- []byte(html)
	})

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	fmt.Printf("Server running on port %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
