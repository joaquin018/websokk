package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
)

// Helper: Formato Moneda CLP ($1.520)
func formatCLP(amount int) string {
	s := strconv.Itoa(amount)
	n := len(s)
	if n <= 3 { return "$" + s }
	res := ""
	for i, r := range s {
		if i > 0 && (n-i)%3 == 0 { res += "." }
		res += string(r)
	}
	return "$" + res
}

// Plantilla Base
const layoutHeader = `
<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Comandas App</title>
    <script src="https://unpkg.com/htmx.org@1.9.11"></script>
    <script src="https://unpkg.com/htmx.org/dist/ext/ws.js"></script>
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        // Helpers para el formateo en tiempo real
        function formatName(el) {
            el.value = el.value.toLowerCase().replace(/\b\w/g, l => l.toUpperCase());
        }
        function formatSentence(el) {
            if (el.value.length > 0) {
                el.value = el.value.charAt(0).toUpperCase() + el.value.slice(1).toLowerCase();
            }
        }
        function formatPrice(el) {
            let val = el.value.replace(/\D/g, "");
            if (val === "") { el.value = ""; return; }
            el.value = "$" + new Intl.NumberFormat("es-CL").format(val);
        }
    </script>
    <style>
        body { background-color: #09090b; color: #fafafa; }
        .kanban-col { min-height: 75vh; }
    </style>
</head>
<body class="bg-zinc-950 text-zinc-50 antialiased overflow-x-hidden">
    <nav class="p-4 border-b border-zinc-800 flex items-center justify-between relative z-50 bg-zinc-950/50 backdrop-blur-md px-6 md:px-12">
        <div class="flex gap-8">
            <a href="/" class="hover:text-blue-400 font-black uppercase text-[10px] tracking-[0.2em] transition-colors">Camareros</a>
            <a href="/cocina" class="hover:text-blue-400 font-black uppercase text-[10px] tracking-[0.2em] transition-colors">Cocina</a>
        </div>
        <a href="/config" class="text-zinc-500 hover:text-white transition-colors p-2 hover:bg-zinc-800 rounded-xl">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/>
            </svg>
        </a>
    </nav>
`

const layoutFooter = `
</body>
</html>
`

// --- VISTA COMANDAS ---
var comandasTmpl = template.Must(template.New("comandas").Parse(layoutHeader + `
    <main class="max-w-md mx-auto p-6 relative">
        <div class="absolute top-40 left-[-50%] -inset-10 bg-blue-500/10 blur-[120px] rounded-full w-96 h-96 opacity-50"></div>
        <div class="absolute bottom-40 right-[-50%] -inset-10 bg-purple-500/10 blur-[120px] rounded-full w-96 h-96 opacity-50"></div>
        <div class="relative z-10 mt-10">
            <h2 class="text-4xl font-bold mb-2 tracking-tighter">Nueva Comanda</h2>
            <p class="text-zinc-500 mb-8 text-sm uppercase font-bold tracking-widest">Panel de Camareros</p>
            <form hx-post="/api/orders" hx-swap="none" hx-on::after-request="this.reset()" class="flex flex-col gap-5">
                <div>
                    <label class="block text-[10px] text-zinc-500 uppercase font-black mb-2 ml-1">Ubicación / Mesa</label>
                    <input type="text" name="mesa" placeholder="Mesa 5" class="w-full bg-zinc-900/50 backdrop-blur-md border border-zinc-800 p-5 rounded-3xl focus:ring-2 focus:ring-blue-500 outline-none transition-all" required>
                </div>
                <div>
                    <label class="block text-[10px] text-zinc-500 uppercase font-black mb-2 ml-1">Qué van a pedir</label>
                    <textarea name="plato" placeholder="Anota aquí..." class="w-full bg-zinc-900/50 backdrop-blur-md border border-zinc-800 p-5 rounded-3xl focus:ring-2 focus:ring-blue-500 outline-none h-40 transition-all" required></textarea>
                </div>
                <button type="submit" class="bg-white text-black hover:bg-zinc-200 py-5 rounded-3xl font-black text-xl shadow-2xl active:scale-95 transition-all mt-4">
                    ENVIAR A COCINA 🚀
                </button>
            </form>
        </div>
    </main>
` + layoutFooter))

// --- VISTA COCINA (TRELLO) ---
var cocinaTmpl = template.Must(template.New("cocina").Parse(layoutHeader + `
    <main class="p-6 relative" hx-ext="ws" ws-connect="/ws">
        <div class="absolute top-40 left-1/4 -inset-10 bg-blue-500/5 blur-3xl rounded-full w-96 h-96 opacity-50"></div>
        <div class="absolute bottom-40 right-1/4 -inset-10 bg-purple-500/5 blur-3xl rounded-full w-96 h-96 opacity-50"></div>
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-8 relative z-10">
            <div class="flex flex-col gap-4">
                <div class="flex items-center gap-3 mb-2"><div class="w-3 h-3 rounded-full bg-yellow-500 animate-pulse"></div><h2 class="font-black text-zinc-400 text-xs uppercase tracking-[0.2em]">Pendientes</h2></div>
                <div id="pedidos-col" class="kanban-col flex flex-col gap-4 p-4 bg-zinc-900/20 backdrop-blur-sm border border-zinc-800/50 rounded-[2rem]"></div>
            </div>
            <div class="flex flex-col gap-4">
                <div class="flex items-center gap-3 mb-2"><div class="w-3 h-3 rounded-full bg-blue-500 animate-pulse"></div><h2 class="font-black text-zinc-400 text-xs uppercase tracking-[0.2em]">En Cocina</h2></div>
                <div id="proceso-col" class="kanban-col flex flex-col gap-4 p-4 bg-zinc-900/20 backdrop-blur-sm border border-zinc-800/50 rounded-[2rem]"></div>
            </div>
            <div class="flex flex-col gap-4">
                <div class="flex items-center gap-3 mb-2"><div class="w-3 h-3 rounded-full bg-green-500 animate-pulse"></div><h2 class="font-black text-zinc-400 text-xs uppercase tracking-[0.2em]">Listos</h2></div>
                <div id="completado-col" class="kanban-col flex flex-col gap-4 p-4 bg-zinc-900/20 backdrop-blur-sm border border-zinc-800/50 rounded-[2rem]"></div>
            </div>
        </div>
    </main>
` + layoutFooter))

// --- VISTA CONFIGURACIÓN ---
var configTmpl = template.Must(template.New("config").Parse(layoutHeader + `
    <main class="max-w-4xl mx-auto p-6 relative">
        <div class="absolute top-20 left-[-20%] bg-blue-500/5 blur-[120px] rounded-full w-96 h-96 opacity-50"></div>
        <div class="relative z-10">
            <h2 class="text-4xl font-bold mb-8 tracking-tighter">Gestión de Productos</h2>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-10">
                <div>
                    <h3 class="text-zinc-500 uppercase text-[10px] font-black tracking-widest mb-4">Añadir Nuevo</h3>
                    <form hx-post="/api/products" hx-target="#product-list" hx-on::after-request="this.reset()" class="flex flex-col gap-4 bg-zinc-900/40 p-6 rounded-[2rem] border border-zinc-800 backdrop-blur-md">
                        <input type="text" name="name" oninput="formatName(this)" placeholder="Nombre del Producto" class="bg-zinc-800/50 border border-zinc-700/50 p-4 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500" required>
                        <input type="text" name="price" oninput="formatPrice(this)" placeholder="Precio (CLP)" class="bg-zinc-800/50 border border-zinc-700/50 p-4 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500" required>
                        <textarea name="description" oninput="formatSentence(this)" placeholder="Descripción corta..." class="bg-zinc-800/50 border border-zinc-700/50 p-4 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500 h-24" required></textarea>
                        <button type="submit" class="bg-blue-600 hover:bg-blue-500 py-4 rounded-2xl font-black uppercase text-xs tracking-widest transition-all">Guardar Producto</button>
                    </form>
                </div>
                <div>
                    <h3 class="text-zinc-500 uppercase text-[10px] font-black tracking-widest mb-4">Productos Actuales</h3>
                    <div id="product-list" hx-get="/api/products" hx-trigger="load" class="flex flex-col gap-3 max-h-[60vh] overflow-y-auto pr-2">
                        <p class="text-zinc-600 text-sm">Cargando productos...</p>
                    </div>
                </div>
            </div>
        </div>
    </main>
` + layoutFooter))

func handleComandas(w http.ResponseWriter, r *http.Request) { comandasTmpl.Execute(w, nil) }
func handleCocina(w http.ResponseWriter, r *http.Request) { cocinaTmpl.Execute(w, nil) }
func handleConfig(w http.ResponseWriter, r *http.Request) { configTmpl.Execute(w, nil) }

func RenderProductList(w io.Writer, products []Product) {
	if len(products) == 0 {
		w.Write([]byte(`<p class="text-zinc-600 text-sm">No hay productos registrados.</p>`))
		return
	}
	for _, p := range products {
		fmt.Fprintf(w, `
			<div class="p-4 bg-zinc-900/60 border border-zinc-800 rounded-2xl flex justify-between items-center animate-in fade-in slide-in-from-right-4 duration-300">
				<div><h4 class="font-bold text-white">%s</h4><p class="text-[10px] text-zinc-500">%s</p></div>
				<span class="font-mono text-blue-400 font-bold text-sm">%s</span>
			</div>`, p.Name, p.Description, formatCLP(p.Price))
	}
}

func RenderOrderCard(id int, mesa, plato, estado string) string {
	btnText, nextStatus, btnClass := "Empezar", "proceso", "bg-zinc-800 hover:bg-zinc-700"
	if estado == "proceso" {
		btnText, nextStatus, btnClass = "Terminar", "completado", "bg-blue-600 hover:bg-blue-500"
	} else if estado == "completado" {
		btnText, nextStatus, btnClass = "Entregado", "delete", "bg-green-600 hover:bg-green-500"
	}
	return fmt.Sprintf(`
		<div id="order-%d" class="p-5 bg-zinc-900/80 backdrop-blur-md border border-zinc-800 rounded-3xl shadow-2xl animate-in fade-in zoom-in slide-in-from-top-4 duration-500">
			<div class="flex justify-between items-start mb-3">
				<span class="text-[10px] font-mono text-zinc-500">ORD-%d</span>
				<span class="px-2.5 py-1 bg-white text-black text-[10px] font-black uppercase rounded-lg">Mesa %s</span>
			</div>
			<h3 class="font-bold text-lg leading-tight mb-4">%s</h3>
			<div class="flex gap-2">
				<button hx-post="/api/orders/update/%d?status=%s" class="flex-1 py-3 text-[10px] font-black uppercase tracking-widest text-white %s rounded-2xl transition-all active:scale-95">%s</button>
			</div>
		</div>`, id, id, mesa, plato, id, nextStatus, btnClass, btnText)
}
