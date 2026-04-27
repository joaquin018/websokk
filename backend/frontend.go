package main

import (
	"fmt"
	"html/template"
	"net/http"
)

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
    <style>
        body { background-color: #09090b; color: #fafafa; }
        .kanban-col { min-height: 75vh; }
    </style>
</head>
<body class="bg-zinc-950 text-zinc-50 antialiased">
    <nav class="p-4 border-b border-zinc-800 flex gap-6 justify-center">
        <a href="/" class="hover:text-blue-400">Inicio</a>
        <a href="/comandas" class="hover:text-blue-400">Camareros</a>
        <a href="/cocina" class="hover:text-blue-400 font-bold text-blue-400">Cocina</a>
    </nav>
`

const layoutFooter = `
</body>
</html>
`

// --- VISTA INICIO ---
var indexTmpl = template.Must(template.New("index").Parse(layoutHeader + `
    <main class="flex min-h-[80vh] flex-col items-center justify-center p-6 text-center">
        <h1 class="text-6xl font-bold tracking-tighter bg-gradient-to-b from-white to-zinc-500 bg-clip-text text-transparent mb-4">Comandas App</h1>
        <p class="text-zinc-400 mb-8">El sistema más rápido para tu restaurante</p>
        <div class="flex gap-4">
            <a href="/comandas" class="px-8 py-4 bg-white text-black font-bold rounded-2xl hover:bg-zinc-200 transition-all">Soy Camarero</a>
            <a href="/cocina" class="px-8 py-4 border border-zinc-800 font-bold rounded-2xl hover:bg-zinc-900 transition-all">Soy Cocina</a>
        </div>
    </main>
` + layoutFooter))

// --- VISTA COMANDAS ---
var comandasTmpl = template.Must(template.New("comandas").Parse(layoutHeader + `
    <main class="max-w-md mx-auto p-6">
        <h2 class="text-3xl font-bold mb-6">Nueva Comanda</h2>
        <form hx-post="/api/orders" hx-swap="none" hx-on::after-request="this.reset()" class="flex flex-col gap-4">
            <div>
                <label class="block text-xs text-zinc-500 uppercase font-bold mb-1 ml-1">Mesa</label>
                <input type="text" name="mesa" placeholder="Ej: 5" class="w-full bg-zinc-900 border border-zinc-800 p-4 rounded-2xl focus:ring-2 focus:ring-blue-500 outline-none" required>
            </div>
            <div>
                <label class="block text-xs text-zinc-500 uppercase font-bold mb-1 ml-1">Pedido</label>
                <textarea name="plato" placeholder="¿Qué van a tomar?" class="w-full bg-zinc-900 border border-zinc-800 p-4 rounded-2xl focus:ring-2 focus:ring-blue-500 outline-none h-32" required></textarea>
            </div>
            <button type="submit" class="bg-blue-600 hover:bg-blue-500 py-4 rounded-2xl font-bold text-lg shadow-lg shadow-blue-900/20 active:scale-95 transition-all mt-2">
                Enviar a Cocina 🚀
            </button>
        </form>
    </main>
` + layoutFooter))

// --- COMPONENTE TARJETA DE PEDIDO ---
func RenderOrderCard(id int, mesa, plato, estado string) string {
	btnText := "Empezar"
	nextStatus := "proceso"

	if estado == "proceso" {
		btnText = "Terminar"
		nextStatus = "completado"
	} else if estado == "completado" {
		btnText = "Eliminar"
		nextStatus = "delete"
	}

	return fmt.Sprintf(`
		<div id="order-%d" class="p-4 bg-zinc-900 border border-zinc-800 rounded-lg shadow-xl animate-in fade-in zoom-in duration-300">
			<div class="flex justify-between items-start mb-2">
				<span class="text-xs font-mono text-zinc-500">#%d</span>
				<span class="px-2 py-0.5 bg-blue-500/10 text-blue-400 text-[10px] font-bold uppercase rounded">Mesa %s</span>
			</div>
			<h3 class="font-bold text-lg">%s</h3>
			<div class="mt-4 flex gap-2">
				<button hx-post="/api/orders/update/%d?status=%s" class="flex-1 py-2 text-xs bg-zinc-800 hover:bg-zinc-700 rounded-xl transition-colors">%s</button>
			</div>
		</div>`, id, id, mesa, plato, id, nextStatus, btnText)
}

// --- VISTA COCINA ---
func handleCocina(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	cocinaTmpl.Execute(w, nil)
}

var cocinaTmpl = template.Must(template.New("cocina").Parse(layoutHeader + `
    <main class="p-6" hx-ext="ws" ws-connect="/ws">
        <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div class="flex flex-col gap-4">
                <div class="flex items-center gap-2 mb-2"><div class="w-2 h-2 rounded-full bg-yellow-500"></div><h2 class="font-bold text-zinc-400 text-sm uppercase">Pendientes</h2></div>
                <div id="pedidos-col" class="kanban-col flex flex-col gap-3 p-3 bg-zinc-900/30 border border-zinc-800/50 rounded-3xl"></div>
            </div>
            <div class="flex flex-col gap-4">
                <div class="flex items-center gap-2 mb-2"><div class="w-2 h-2 rounded-full bg-blue-500"></div><h2 class="font-bold text-zinc-400 text-sm uppercase">En Cocina</h2></div>
                <div id="proceso-col" class="kanban-col flex flex-col gap-3 p-3 bg-zinc-900/30 border border-zinc-800/50 rounded-3xl"></div>
            </div>
            <div class="flex flex-col gap-4">
                <div class="flex items-center gap-2 mb-2"><div class="w-2 h-2 rounded-full bg-green-500"></div><h2 class="font-bold text-zinc-400 text-sm uppercase">Listos</h2></div>
                <div id="completado-col" class="kanban-col flex flex-col gap-3 p-3 bg-zinc-900/30 border border-zinc-800/50 rounded-3xl"></div>
            </div>
        </div>
    </main>
` + layoutFooter))

func handleIndex(w http.ResponseWriter, r *http.Request) { indexTmpl.Execute(w, nil) }
func handleComandas(w http.ResponseWriter, r *http.Request) { comandasTmpl.Execute(w, nil) }
