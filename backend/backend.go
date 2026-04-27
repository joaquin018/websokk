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
		defer db.Close(context.Background())
	}

	hub := newHub()
	go hub.run()

	// --- RUTAS ---
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if db == nil { initDB() }
		rows, _ := db.Query(context.Background(), "SELECT id, name FROM tables ORDER BY name ASC")
		var tables []Table
		if rows != nil {
			for rows.Next() {
				var t Table
				rows.Scan(&t.ID, &t.Name)
				tables = append(tables, t)
			}
		}
		RenderComandasPage(w, tables)
	})
	http.HandleFunc("/cocina", func(w http.ResponseWriter, r *http.Request) {
		if db == nil { initDB() }
		if db == nil {
			http.Error(w, "Base de datos no disponible", http.StatusServiceUnavailable)
			return
		}
		rows, err := db.Query(context.Background(), "SELECT id, plato, mesa, estado FROM orders ORDER BY id ASC")
		if err != nil {
			http.Error(w, fmt.Sprintf("Error al consultar la DB: %v", err), http.StatusInternalServerError)
			return
		}
		var orders []Order
		for rows.Next() {
			var o Order
			rows.Scan(&o.ID, &o.Plato, &o.Mesa, &o.Estado)
			orders = append(orders, o)
		}
		RenderCocinaPage(w, orders)
	})
	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		if db == nil { initDB() }
		rows, _ := db.Query(context.Background(), "SELECT id, name FROM tables ORDER BY name ASC")
		var tables []Table
		if rows != nil {
			for rows.Next() {
				var t Table
				rows.Scan(&t.ID, &t.Name)
				tables = append(tables, t)
			}
		}
		RenderConfigPage(w, tables)
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
			
			html := fmt.Sprintf(`<div id="pedidos-col" hx-swap-oob="beforeend">%s</div>`, RenderOrderCard(id, mesa, plato, "pendiente"))
			hub.broadcast <- []byte(html)
			
			w.Write([]byte(`
				<form hx-post="/api/orders" hx-swap="none" hx-on::after-request="this.reset()" class="flex flex-col gap-6 bg-zinc-900/20 p-6 md:p-8 rounded-[2rem] border border-zinc-800/50 backdrop-blur-sm">
					<div>
						<label class="block text-[10px] text-zinc-500 uppercase font-black mb-3 ml-1 tracking-widest">Ubicación / Mesa</label>
						<input type="text" name="mesa" placeholder="Ej: Mesa 12" class="w-full bg-zinc-950/50 border border-zinc-800 p-4 md:p-5 rounded-2xl md:rounded-3xl focus:ring-2 focus:ring-blue-500 outline-none transition-all text-xl font-bold" required>
					</div>
					<div>
						<label class="block text-[10px] text-zinc-500 uppercase font-black mb-3 ml-1 tracking-widest">Pedido Detallado</label>
						<textarea id="order-text" name="plato" placeholder="Los productos seleccionados aparecerán aquí..." class="w-full bg-zinc-950/50 border border-zinc-800 p-4 md:p-5 rounded-2xl md:rounded-3xl focus:ring-2 focus:ring-blue-500 outline-none h-48 transition-all font-medium text-lg leading-relaxed" required></textarea>
					</div>
					<button type="submit" class="group relative overflow-hidden bg-white text-black py-5 md:py-6 rounded-2xl md:rounded-3xl font-black text-xl shadow-2xl active:scale-95 transition-all mt-4">
						<span class="relative z-10 flex items-center justify-center gap-3">ENVIAR A COCINA <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m5 12 7-7 7 7"/><path d="M12 19V5"/></svg></span>
					</button>
				</form>`))
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
		
		targetCol := "proceso-col"
		if newStatus == "completado" { targetCol = "completado-col" }
		
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
