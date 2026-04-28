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
	initDB()
	if db != nil {
		defer db.Close()
	}

	hub := newHub()
	go hub.run()
	
	// --- SERVICE WORKER ---
	http.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(`
			const CACHE_NAME = 'comandas-v2';
			const ASSETS = [
				'/',
				'https://unpkg.com/htmx.org@1.9.11',
				'https://cdn.tailwindcss.com'
			];

			self.addEventListener('install', e => {
				e.waitUntil(caches.open(CACHE_NAME).then(c => c.addAll(ASSETS)));
			});

			self.addEventListener('fetch', e => {
				e.respondWith(
					caches.match(e.request).then(res => {
						return res || fetch(e.request);
					})
				);
			});
		`))
	})

	// --- RUTA PRINCIPAL UNIFICADA (Zero Latency) ---
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if db == nil { initDB() }
		
		// 1. Obtener Mesas
		rowsT, _ := db.Query(context.Background(), "SELECT id, name FROM tables ORDER BY name ASC")
		var tables []Table
		if rowsT != nil {
			defer rowsT.Close()
			for rowsT.Next() {
				var t Table
				rowsT.Scan(&t.ID, &t.Name)
				tables = append(tables, t)
			}
		}

		// 2. Obtener Pedidos (Cocina)
		rowsO, _ := db.Query(context.Background(), "SELECT id, plato, mesa, estado FROM orders ORDER BY id ASC")
		var orders []Order
		if rowsO != nil {
			defer rowsO.Close()
			for rowsO.Next() {
				var o Order
				rowsO.Scan(&o.ID, &o.Plato, &o.Mesa, &o.Estado)
				orders = append(orders, o)
			}
		}

		// 3. Obtener Productos (Config)
		rowsP, _ := db.Query(context.Background(), "SELECT id, name, price, description FROM products ORDER BY id DESC")
		var products []Product
		if rowsP != nil {
			defer rowsP.Close()
			for rowsP.Next() {
				var p Product
				rowsP.Scan(&p.ID, &p.Name, &p.Price, &p.Description)
				products = append(products, p)
			}
		}

		RenderMainPage(w, tables, orders, products)
	})
	
	http.HandleFunc("/cocina", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	
	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

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

	// --- API PRODUCTOS (CRUD) ---
	http.HandleFunc("/api/products", func(w http.ResponseWriter, r *http.Request) {
		if db == nil { initDB() }
		if db == nil { return }
		if r.Method == http.MethodPost {
			name := r.FormValue("name")
			description := r.FormValue("description")
			priceRaw := r.FormValue("price")
			priceStr := ""
			for _, char := range priceRaw {
				if unicode.IsDigit(char) { priceStr += string(char) }
			}
			price, _ := strconv.Atoi(priceStr)
			_, _ = db.Exec(context.Background(), "INSERT INTO products (name, price, description) VALUES ($1, $2, $3)", name, price, description)
		}
		rows, _ := db.Query(context.Background(), "SELECT id, name, price, description FROM products ORDER BY id DESC")
		var products []Product
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var p Product
				rows.Scan(&p.ID, &p.Name, &p.Price, &p.Description)
				products = append(products, p)
			}
		}
		RenderProductList(w, products)
	})

	http.HandleFunc("/api/products/delete/", func(w http.ResponseWriter, r *http.Request) {
		if db == nil { return }
		idStr := strings.TrimPrefix(r.URL.Path, "/api/products/delete/")
		id, _ := strconv.Atoi(idStr)
		_, _ = db.Exec(context.Background(), "DELETE FROM products WHERE id=$1", id)
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/products/edit/", func(w http.ResponseWriter, r *http.Request) {
		if db == nil { return }
		idStr := strings.TrimPrefix(r.URL.Path, "/api/products/edit/")
		id, _ := strconv.Atoi(idStr)
		var p Product
		_ = db.QueryRow(context.Background(), "SELECT id, name, price, description FROM products WHERE id=$1", id).Scan(&p.ID, &p.Name, &p.Price, &p.Description)
		RenderProductEditForm(w, p)
	})

	http.HandleFunc("/api/products/update/", func(w http.ResponseWriter, r *http.Request) {
		if db == nil { return }
		idStr := strings.TrimPrefix(r.URL.Path, "/api/products/update/")
		id, _ := strconv.Atoi(idStr)
		name := r.FormValue("name")
		description := r.FormValue("description")
		priceRaw := r.FormValue("price")
		priceStr := ""
		for _, char := range priceRaw {
			if unicode.IsDigit(char) { priceStr += string(char) }
		}
		price, _ := strconv.Atoi(priceStr)
		
		var p Product
		_ = db.QueryRow(context.Background(), "UPDATE products SET name=$1, price=$2, description=$3 WHERE id=$4 RETURNING id, name, price, description", name, price, description, id).Scan(&p.ID, &p.Name, &p.Price, &p.Description)
		RenderProductItem(w, p)
	})

	http.HandleFunc("/api/products/search", func(w http.ResponseWriter, r *http.Request) {
		if db == nil { initDB() }
		if db == nil { return }
		q := r.URL.Query().Get("q")
		if q == "" { return }
		rows, _ := db.Query(context.Background(), "SELECT name, price FROM products WHERE name ILIKE $1 LIMIT 5", "%"+q+"%")
		var products []Product
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var p Product
				rows.Scan(&p.Name, &p.Price)
				products = append(products, p)
			}
		}
		RenderSearchSuggestions(w, products)
	})

	// --- API TABLES (GENERACIÓN AUTOMÁTICA) ---
	http.HandleFunc("/api/tables", func(w http.ResponseWriter, r *http.Request) {
		if db == nil { initDB() }
		if r.Method == http.MethodPost {
			countStr := r.FormValue("count")
			count, _ := strconv.Atoi(countStr)
			
			// Borramos las anteriores y creamos las nuevas
			_, _ = db.Exec(context.Background(), "DELETE FROM tables")
			for i := 1; i <= count; i++ {
				name := fmt.Sprintf("Mesa %d", i)
				_, _ = db.Exec(context.Background(), "INSERT INTO tables (name) VALUES ($1)", name)
			}
		}
		rows, _ := db.Query(context.Background(), "SELECT id, name FROM tables ORDER BY id ASC")
		var tables []Table
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var t Table
				rows.Scan(&t.ID, &t.Name)
				tables = append(tables, t)
			}
		}
		RenderTableList(w, tables)
	})

	http.HandleFunc("/api/tables/delete/", func(w http.ResponseWriter, r *http.Request) {
		if db == nil { return }
		idStr := strings.TrimPrefix(r.URL.Path, "/api/tables/delete/")
		id, _ := strconv.Atoi(idStr)
		_, _ = db.Exec(context.Background(), "DELETE FROM tables WHERE id=$1", id)
		w.WriteHeader(http.StatusOK)
	})

	// --- API PEDIDOS ---
	http.HandleFunc("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		if db == nil { initDB() }
		if db == nil { return }
		if r.Method == http.MethodPost {
			plato, mesa := r.FormValue("plato"), r.FormValue("mesa")
			var id int
			err := db.QueryRow(context.Background(), "INSERT INTO orders (plato, mesa, estado) VALUES ($1, $2, 'pendiente') RETURNING id", plato, mesa).Scan(&id)
			if err != nil { return }
			
			html := fmt.Sprintf(`<div id="col-pendiente" hx-swap-oob="beforeend">%s</div>`, RenderOrderCard(id, mesa, plato, "pendiente"))
			hub.broadcast <- []byte(html)
			
			w.WriteHeader(http.StatusOK)
		}
	})

	http.HandleFunc("/api/orders/update/", func(w http.ResponseWriter, r *http.Request) {
		if db == nil { initDB() }
		if db == nil { return }
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
		
		targetCol := "col-proceso"
		if newStatus == "completado" { targetCol = "col-completado" }
		
		html := fmt.Sprintf(`
			<div id="order-%d" hx-swap-oob="delete"></div>
			<div id="%s" hx-swap-oob="beforeend">%s</div>`, id, targetCol, RenderOrderCard(id, mesa, plato, newStatus))
		
		hub.broadcast <- []byte(html)
	})

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	fmt.Printf("Server running on port %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
