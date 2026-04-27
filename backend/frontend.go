package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
)

// Helper: Formato Moneda CLP
func formatCLP(amount int) string {
	s := strconv.Itoa(amount)
	n := len(s)
	if n <= 3 {
		return "$" + s
	}
	res := ""
	for i, r := range s {
		if i > 0 && (n-i)%3 == 0 {
			res += "."
		}
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
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>Comandas App</title>
    <script src="https://unpkg.com/htmx.org@1.9.11"></script>
    <script src="https://unpkg.com/htmx.org/dist/ext/ws.js"></script>
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        function formatName(el) { el.value = el.value.toLowerCase().replace(/\b\w/g, l => l.toUpperCase()); }
        function formatSentence(el) { if (el.value.length > 0) el.value = el.value.charAt(0).toUpperCase() + el.value.slice(1).toLowerCase(); }
        function formatPrice(el) {
            let val = el.value.replace(/\D/g, "");
            if (val === "") { el.value = ""; return; }
            el.value = "$" + new Intl.NumberFormat("es-CL").format(val);
        }
        function addProductToOrder(name) {
            const textarea = document.getElementById('order-text');
            if (textarea.value.length > 0) textarea.value += "\n";
            textarea.value += name;
            textarea.classList.add('ring-4', 'ring-blue-500/50');
            setTimeout(() => textarea.classList.remove('ring-4', 'ring-blue-500/50'), 400);
            document.getElementById('search-results').innerHTML = '';
            document.getElementById('product-search').value = '';
            textarea.focus();
        }
        function selectTable(name, btn) {
            document.getElementById('mesa-input').value = name;
            document.getElementById('selected-mesa-display').innerText = name;
            
            // Transición visual: Ocultar grid de mesas, mostrar formulario de pedido
            document.getElementById('view-tables').classList.add('hidden');
            document.getElementById('view-order').classList.remove('hidden');
            document.getElementById('view-order').classList.add('animate-in', 'fade-in', 'slide-in-from-bottom-10', 'duration-500');
            
            // Hacer scroll al inicio por si acaso
            window.scrollTo({ top: 0, behavior: 'smooth' });
        }
        function backToTables() {
            document.getElementById('view-tables').classList.remove('hidden');
            document.getElementById('view-order').classList.add('hidden');
            document.getElementById('view-tables').classList.add('animate-in', 'fade-in', 'slide-in-from-top-10', 'duration-500');
        }
    </script>
    <style>
        body { background-color: #09090b; color: #fafafa; -webkit-tap-highlight-color: transparent; }
        .kanban-col { min-height: 200px; }
        @media (min-width: 1024px) {
            .kanban-col { min-height: 75vh; }
        }
        ::-webkit-scrollbar { width: 6px; height: 6px; }
        ::-webkit-scrollbar-track { background: transparent; }
        ::-webkit-scrollbar-thumb { background: #27272a; border-radius: 10px; }
        ::-webkit-scrollbar-thumb:hover { background: #3f3f46; }
    </style>
</head>
<body class="bg-zinc-950 text-zinc-50 antialiased overflow-x-hidden">
    <nav class="sticky top-0 z-50 bg-zinc-950/80 backdrop-blur-xl border-b border-zinc-800/50 px-4 md:px-12 py-3">
        <div class="max-w-7xl mx-auto flex items-center justify-end">
            <!-- Ajustes -->
            <a href="/config" class="text-zinc-500 hover:text-white transition-all p-2 hover:bg-zinc-800 rounded-2xl border border-transparent hover:border-zinc-700">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/>
                </svg>
            </a>
        </div>
    </nav>
`

const layoutFooter = `
</body>
</html>
`

// --- VISTA COMANDAS ---
var comandasTmpl = template.Must(template.New("comandas").Parse(layoutHeader + `
    <main class="max-w-6xl mx-auto p-4 md:p-8 pb-32 relative">
        <!-- VISTA 1: SELECCIÓN DE MESA -->
        <div id="view-tables" class="relative z-10 mt-4 md:mt-10">
            <header class="mb-12 text-center">
                <h2 class="text-3xl md:text-5xl font-black mb-2 tracking-tighter">Seleccionar Mesa</h2>
                <p class="text-zinc-500 text-[10px] md:text-xs uppercase font-bold tracking-[0.2em]">Toca una mesa para empezar el pedido</p>
            </header>

            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4 md:gap-6">
                {{range .Tables}}
                <button type="button" onclick="selectTable('{{.Name}}', this)" class="group flex flex-col items-center justify-center gap-6 aspect-video md:aspect-square bg-zinc-900/40 border border-zinc-800/50 rounded-[2.5rem] transition-all hover:bg-blue-600/10 hover:border-blue-500/50 active:scale-95">
                    <div class="p-4 bg-zinc-950/50 rounded-2xl group-hover:bg-blue-500 group-hover:text-white transition-all text-zinc-600">
                        <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3v18"/><rect width="18" height="12" x="3" y="6" rx="2"/></svg>
                    </div>
                    <span class="font-black text-xl md:text-2xl tracking-tighter text-zinc-300 group-hover:text-white">{{.Name}}</span>
                </button>
                {{else}}
                <div class="col-span-full p-20 border-2 border-dashed border-zinc-900 rounded-[3rem] text-center">
                    <p class="text-zinc-700 font-black uppercase tracking-[0.3em] mb-4">No hay mesas configuradas</p>
                    <a href="/config" class="text-blue-500 font-bold hover:underline">Ir a configuración</a>
                </div>
                {{end}}
            </div>
        </div>

        <!-- VISTA 2: BUSCADOR Y PEDIDO -->
        <div id="view-order" class="hidden relative z-10 mt-4 md:mt-10">
            <header class="mb-8 flex items-center justify-between">
                <button onclick="backToTables()" class="p-4 bg-zinc-900/60 border border-zinc-800 rounded-2xl text-zinc-400 hover:text-white transition-all">
                    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>
                </button>
                <div class="text-center">
                    <p class="text-zinc-500 text-[10px] md:text-xs uppercase font-bold tracking-[0.2em]">Pedido para</p>
                    <h2 id="selected-mesa-display" class="text-3xl md:text-4xl font-black tracking-tighter text-blue-500">Mesa X</h2>
                </div>
                <div class="w-14"></div> <!-- Spacer -->
            </header>

            <div class="max-w-2xl mx-auto flex flex-col gap-8">
                <div class="relative">
                    <label class="block text-[10px] text-zinc-500 uppercase font-black mb-3 ml-1 tracking-widest">Buscador Rápido</label>
                    <div class="relative group">
                        <input type="text" id="product-search" name="q" hx-get="/api/products/search" hx-trigger="keyup changed delay:300ms" hx-target="#search-results" placeholder="Escribe para buscar..." class="w-full bg-zinc-900/40 backdrop-blur-xl border border-zinc-800 p-4 md:p-6 rounded-2xl md:rounded-3xl focus:ring-2 focus:ring-blue-500 outline-none transition-all placeholder:text-zinc-700 text-lg">
                        <div class="absolute right-4 top-1/2 -translate-y-1/2 text-zinc-700 group-focus-within:text-blue-500 transition-colors">
                            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
                        </div>
                    </div>
                    <div id="search-results" class="absolute w-full mt-3 z-[60] flex flex-col gap-2 drop-shadow-2xl"></div>
                </div>

                <form hx-post="/api/orders" hx-swap="none" hx-on::after-request="this.reset(); backToTables()" class="flex flex-col gap-6 bg-zinc-900/20 p-6 md:p-8 rounded-[2rem] border border-zinc-800/50 backdrop-blur-sm">
                    <input type="hidden" name="mesa" id="mesa-input">
                    <div>
                        <label class="block text-[10px] text-zinc-500 uppercase font-black mb-3 ml-1 tracking-widest">Detalles del Pedido</label>
                        <textarea id="order-text" name="plato" placeholder="Los productos seleccionados aparecerán aquí..." class="w-full bg-zinc-950/50 border border-zinc-800 p-4 md:p-5 rounded-2xl md:rounded-3xl focus:ring-2 focus:ring-blue-500 outline-none h-48 transition-all font-medium text-lg leading-relaxed" required></textarea>
                    </div>
                    <button type="submit" class="group relative overflow-hidden bg-white text-black py-5 md:py-6 rounded-2xl md:rounded-3xl font-black text-xl shadow-2xl active:scale-95 transition-all">
                        <span class="relative z-10 flex items-center justify-center gap-3 uppercase">Confirmar y Enviar <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m5 12 7-7 7 7"/><path d="M12 19V5"/></svg></span>
                    </button>
                </form>
            </div>
        </div>
    </main>

    <!-- NAVEGACIÓN INFERIOR (Estilo App) -->
    <nav class="fixed bottom-6 left-1/2 -translate-x-1/2 z-[100] w-[90%] max-w-sm">
        <div class="bg-zinc-900/80 backdrop-blur-2xl border border-zinc-800/50 p-2 rounded-[2.5rem] flex items-center justify-around shadow-2xl shadow-black">
            <a href="/" class="flex-1 flex flex-col items-center gap-1 py-3 px-6 rounded-[2rem] transition-all bg-blue-600 text-white">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 12h6"/><path d="M9 16h6"/><path d="M9 8h6"/></svg>
                <span class="text-[9px] font-black uppercase tracking-widest">Comandas</span>
            </a>
            <a href="/cocina" class="flex-1 flex flex-col items-center gap-1 py-3 px-6 rounded-[2rem] transition-all text-zinc-500 hover:text-white">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z"/></svg>
                <span class="text-[9px] font-black uppercase tracking-widest">Cocina</span>
            </a>
        </div>
    </nav>
` + layoutFooter))

// --- VISTA CONFIGURACIÓN ---
var configTmpl = template.Must(template.New("config").Parse(`
    <main class="max-w-6xl mx-auto p-4 md:p-8 relative">
        <div class="absolute top-20 left-[-10%] bg-blue-500/5 blur-[120px] rounded-full w-96 h-96 opacity-30"></div>
        
        <div class="grid grid-cols-1 lg:grid-cols-12 gap-12 relative z-10">
            <!-- Gestión de Mesas -->
            <div class="lg:col-span-4 flex flex-col gap-8">
                <div>
                    <h3 class="text-zinc-500 uppercase text-[10px] font-black tracking-widest mb-6 ml-2">Capacidad del Local</h3>
                    <form hx-post="/api/tables" hx-target="#table-list" class="flex flex-col gap-4 bg-zinc-900/40 p-6 rounded-[2rem] border border-zinc-800/50 backdrop-blur-md">
                        <label class="text-[10px] text-zinc-500 uppercase font-black ml-1 tracking-widest">¿Cuántas mesas tienes?</label>
                        <input type="number" name="count" min="1" max="50" placeholder="Ej: 10" class="bg-zinc-950/50 border border-zinc-800 p-4 rounded-xl outline-none focus:ring-2 focus:ring-blue-500 transition-all text-2xl font-black text-center" required>
                        <button type="submit" class="bg-zinc-100 text-black hover:bg-white py-4 rounded-xl font-black uppercase text-[10px] tracking-widest active:scale-95 transition-all">Generar Mesas Automáticamente</button>
                    </form>
                </div>
                <div id="table-list" class="grid grid-cols-2 gap-3 max-h-[40vh] overflow-y-auto pr-2 custom-scrollbar">
                    {{range .Tables}}
                        <div class="p-3 bg-zinc-900/60 border border-zinc-800 rounded-xl text-center">
                            <span class="font-bold text-zinc-400 text-xs">{{.Name}}</span>
                        </div>
                    {{end}}
                </div>
            </div>

            <!-- Gestión del Menú -->
            <div class="lg:col-span-8 flex flex-col gap-8">
                <div>
                    <h3 class="text-zinc-500 uppercase text-[10px] font-black tracking-widest mb-6 ml-2">Nuevo Producto</h3>
                    <form hx-post="/api/products" hx-target="#product-list" hx-on::after-request="this.reset()" class="grid grid-cols-1 md:grid-cols-2 gap-5 bg-zinc-900/40 p-6 md:p-8 rounded-[2.5rem] border border-zinc-800/50 backdrop-blur-md">
                        <div class="flex flex-col gap-4">
                            <input type="text" name="name" oninput="formatName(this)" placeholder="Nombre del Producto" class="bg-zinc-950/50 border border-zinc-800 p-4 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500 transition-all" required>
                            <input type="text" name="price" oninput="formatPrice(this)" placeholder="Precio (CLP)" class="bg-zinc-950/50 border border-zinc-800 p-4 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500 transition-all" required>
                        </div>
                        <textarea name="description" oninput="formatSentence(this)" placeholder="Descripción breve..." class="bg-zinc-950/50 border border-zinc-800 p-4 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500 h-full min-h-[120px] transition-all" required></textarea>
                        <button type="submit" class="md:col-span-2 bg-blue-600 hover:bg-blue-500 py-5 rounded-2xl font-black uppercase text-xs tracking-widest shadow-xl shadow-blue-900/20 active:scale-95 transition-all">Guardar en el Menú</button>
                    </form>
                </div>
                <div id="product-list" hx-get="/api/products" hx-trigger="load" class="grid grid-cols-1 gap-4 max-h-[60vh] overflow-y-auto pr-2 custom-scrollbar">
                    <div class="p-8 text-center border-2 border-dashed border-zinc-900 rounded-[2rem] text-zinc-700 font-bold uppercase text-[10px] tracking-widest">Cargando inventario...</div>
                </div>
            </div>
        </div>
    </main>
`))

func RenderComandasPage(w http.ResponseWriter, tables []Table) {
	comandasTmpl.Execute(w, map[string]interface{}{"Tables": tables})
}

func RenderConfigPage(w http.ResponseWriter, tables []Table) {
	w.Write([]byte(`
<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Configuraciones</title>
    <script src="https://unpkg.com/htmx.org@1.9.11"></script>
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        function formatName(el) { el.value = el.value.toLowerCase().replace(/\b\w/g, l => l.toUpperCase()); }
        function formatSentence(el) { if (el.value.length > 0) el.value = el.value.charAt(0).toUpperCase() + el.value.slice(1).toLowerCase(); }
        function formatPrice(el) {
            let val = el.value.replace(/\D/g, "");
            if (val === "") { el.value = ""; return; }
            el.value = "$" + new Intl.NumberFormat("es-CL").format(val);
        }
    </script>
    <style>
        body { background-color: #09090b; color: #fafafa; }
        ::-webkit-scrollbar { width: 6px; height: 6px; }
        ::-webkit-scrollbar-track { background: transparent; }
        ::-webkit-scrollbar-thumb { background: #27272a; border-radius: 10px; }
        ::-webkit-scrollbar-thumb:hover { background: #3f3f46; }
    </style>
</head>
<body class="bg-zinc-950 text-zinc-50 antialiased">
    <nav class="sticky top-0 z-50 bg-zinc-950/80 backdrop-blur-xl border-b border-zinc-800/50 px-4 md:px-12 py-4">
        <div class="max-w-7xl mx-auto flex items-center gap-6">
            <a href="/" class="p-2 hover:bg-zinc-800 rounded-xl transition-all text-zinc-400 hover:text-white">
                <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>
            </a>
            <h1 class="text-xl md:text-2xl font-bold tracking-tight">Configuraciones</h1>
        </div>
    </nav>
`))
	configTmpl.Execute(w, map[string]interface{}{"Tables": tables})
	w.Write([]byte(layoutFooter))
}

func RenderTableList(w io.Writer, tables []Table) {
	for _, t := range tables {
		RenderTableItem(w, t)
	}
}

func RenderTableItem(w io.Writer, t Table) {
	fmt.Fprintf(w, `
		<div class="p-3 bg-zinc-900/60 border border-zinc-800 rounded-xl text-center animate-in zoom-in duration-200">
			<span class="font-bold text-zinc-400 text-xs">%s</span>
		</div>`, t.Name)
}

func handleComandas(w http.ResponseWriter, r *http.Request) { RenderComandasPage(w, nil) }
func handleConfig(w http.ResponseWriter, r *http.Request)   { RenderConfigPage(w, nil) }

func RenderCocinaPage(w http.ResponseWriter, orders []Order) {
	w.Write([]byte(layoutHeader))
	w.Write([]byte(`
    <main class="p-4 md:p-8 relative" hx-ext="ws" ws-connect="/ws">
        <div class="absolute top-40 left-1/4 -inset-10 bg-blue-500/5 blur-3xl rounded-full w-96 h-96 opacity-20"></div>
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-6 md:gap-8 relative z-10">
            <div class="flex flex-col gap-5">
                <div class="flex items-center gap-3 px-4"><div class="w-2 h-2 rounded-full bg-yellow-500 animate-pulse"></div><h2 class="font-black text-zinc-400 text-[10px] uppercase tracking-[0.2em]">Pendientes</h2></div>
                <div id="pedidos-col" class="kanban-col flex flex-col gap-4 p-3 md:p-5 bg-zinc-900/20 backdrop-blur-md border border-zinc-800/50 rounded-[2.5rem] md:rounded-[3rem]">`))
	for _, o := range orders {
		if o.Estado == "pendiente" {
			w.Write([]byte(RenderOrderCard(o.ID, o.Mesa, o.Plato, o.Estado)))
		}
	}
	w.Write([]byte(`</div>
            </div>
            <div class="flex flex-col gap-5">
                <div class="flex items-center gap-3 px-4"><div class="w-2 h-2 rounded-full bg-blue-500 animate-pulse"></div><h2 class="font-black text-zinc-400 text-[10px] uppercase tracking-[0.2em]">En Proceso</h2></div>
                <div id="proceso-col" class="kanban-col flex flex-col gap-4 p-3 md:p-5 bg-zinc-900/20 backdrop-blur-md border border-zinc-800/50 rounded-[2.5rem] md:rounded-[3rem]">`))
	for _, o := range orders {
		if o.Estado == "proceso" {
			w.Write([]byte(RenderOrderCard(o.ID, o.Mesa, o.Plato, o.Estado)))
		}
	}
	w.Write([]byte(`</div>
            </div>
            <div class="flex flex-col gap-5">
                <div class="flex items-center gap-3 px-4"><div class="w-2 h-2 rounded-full bg-green-500 animate-pulse"></div><h2 class="font-black text-zinc-400 text-[10px] uppercase tracking-[0.2em]">Completado</h2></div>
                <div id="completado-col" class="kanban-col flex flex-col gap-4 p-3 md:p-5 bg-zinc-900/20 backdrop-blur-md border border-zinc-800/50 rounded-[2.5rem] md:rounded-[3rem]">`))
	for _, o := range orders {
		if o.Estado == "completado" {
			w.Write([]byte(RenderOrderCard(o.ID, o.Mesa, o.Plato, o.Estado)))
		}
	}
	w.Write([]byte(`</div>
            </div>
        </div>
    </main>

    <!-- NAVEGACIÓN INFERIOR (Estilo App) -->
    <nav class="fixed bottom-6 left-1/2 -translate-x-1/2 z-[100] w-[90%] max-w-sm">
        <div class="bg-zinc-900/80 backdrop-blur-2xl border border-zinc-800/50 p-2 rounded-[2.5rem] flex items-center justify-around shadow-2xl shadow-black">
            <a href="/" class="flex-1 flex flex-col items-center gap-1 py-3 px-6 rounded-[2rem] transition-all text-zinc-500 hover:text-white">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 12h6"/><path d="M9 16h6"/><path d="M9 8h6"/></svg>
                <span class="text-[9px] font-black uppercase tracking-widest">Comandas</span>
            </a>
            <a href="/cocina" class="flex-1 flex flex-col items-center gap-1 py-3 px-6 rounded-[2rem] transition-all bg-blue-600 text-white">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z"/></svg>
                <span class="text-[9px] font-black uppercase tracking-widest">Cocina</span>
            </a>
        </div>
    </nav>
`))
	w.Write([]byte(layoutFooter))
}

func RenderProductList(w io.Writer, products []Product) {
	if len(products) == 0 {
		w.Write([]byte(`<div class="p-12 text-center border-2 border-dashed border-zinc-900 rounded-[2rem] text-zinc-700 font-bold uppercase text-[10px] tracking-widest">Sin productos</div>`))
		return
	}
	for _, p := range products {
		RenderProductItem(w, p)
	}
}

func RenderProductItem(w io.Writer, p Product) {
	fmt.Fprintf(w, `
		<div id="product-%d" class="p-5 md:p-6 bg-zinc-900/60 border border-zinc-800/50 rounded-3xl flex justify-between items-center animate-in fade-in slide-in-from-right-4 duration-300 group hover:border-zinc-700 transition-all">
			<div class="flex flex-col gap-1">
				<h4 class="font-black text-white text-base md:text-lg group-hover:text-blue-400 transition-colors">%s</h4>
				<p class="text-[10px] text-zinc-500 uppercase tracking-widest font-bold">%s</p>
				<span class="font-black text-blue-500 text-lg md:text-xl tracking-tighter">%s</span>
			</div>
			<div class="flex gap-2">
				<button hx-get="/api/products/edit/%d" hx-target="#product-%d" hx-swap="outerHTML" class="p-3 bg-zinc-800 hover:bg-zinc-700 text-zinc-400 hover:text-white rounded-xl transition-all">
					<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/><path d="m15 5 4 4"/></svg>
				</button>
				<button hx-delete="/api/products/delete/%d" hx-target="#product-%d" hx-swap="outerHTML" hx-confirm="¿Seguro que quieres eliminar %s?" class="p-3 bg-zinc-800 hover:bg-red-900/50 text-zinc-400 hover:text-red-400 rounded-xl transition-all">
					<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg>
				</button>
			</div>
		</div>`, p.ID, p.Name, p.Description, formatCLP(p.Price), p.ID, p.ID, p.ID, p.ID, p.Name)
}

func RenderProductEditForm(w io.Writer, p Product) {
	fmt.Fprintf(w, `
		<form id="product-%d" hx-post="/api/products/update/%d" hx-target="#product-%d" hx-swap="outerHTML" class="p-5 md:p-6 bg-zinc-800/40 border-2 border-blue-500/30 rounded-3xl flex flex-col gap-4 animate-in zoom-in duration-200">
			<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
				<input type="text" name="name" value="%s" oninput="formatName(this)" class="bg-zinc-950/50 border border-zinc-700 p-3 rounded-xl outline-none focus:ring-2 focus:ring-blue-500 text-sm font-bold" required>
				<input type="text" name="price" value="%s" oninput="formatPrice(this)" class="bg-zinc-950/50 border border-zinc-700 p-3 rounded-xl outline-none focus:ring-2 focus:ring-blue-500 text-sm font-bold" required>
			</div>
			<textarea name="description" oninput="formatSentence(this)" class="bg-zinc-950/50 border border-zinc-700 p-3 rounded-xl outline-none focus:ring-2 focus:ring-blue-500 text-xs h-20" required>%s</textarea>
			<div class="flex gap-2">
				<button type="submit" class="flex-1 bg-blue-600 hover:bg-blue-500 py-3 rounded-xl font-black uppercase text-[10px] tracking-widest transition-all">Guardar Cambios</button>
				<button type="button" hx-get="/api/products" hx-target="#product-list" class="px-6 bg-zinc-800 hover:bg-zinc-700 py-3 rounded-xl font-black uppercase text-[10px] tracking-widest transition-all">Cancelar</button>
			</div>
		</form>`, p.ID, p.ID, p.ID, p.Name, formatCLP(p.Price), p.Description)
}

func RenderSearchSuggestions(w io.Writer, products []Product) {
	if len(products) == 0 {
		w.Write([]byte(`<div class="p-6 bg-zinc-900 border border-zinc-800 rounded-3xl text-zinc-500 text-xs font-bold uppercase tracking-widest text-center">Sin resultados</div>`))
		return
	}
	for _, p := range products {
		fmt.Fprintf(w, `
			<button type="button" onclick="addProductToOrder('%s')" class="w-full p-5 md:p-6 bg-zinc-900/90 backdrop-blur-xl hover:bg-blue-600 border border-zinc-800 text-left rounded-3xl transition-all group animate-in fade-in slide-in-from-top-2 duration-200">
				<div class="flex justify-between items-center">
					<div class="flex flex-col">
                        <span class="font-black text-white text-lg group-hover:text-white">%s</span>
                        <span class="text-[10px] text-zinc-500 group-hover:text-blue-200 uppercase tracking-widest font-bold">Añadir al pedido</span>
                    </div>
					<span class="text-xl font-black text-blue-500 group-hover:text-white tracking-tighter">%s</span>
				</div>
			</button>`, p.Name, p.Name, formatCLP(p.Price))
	}
}

func RenderOrderCard(id int, mesa, plato, estado string) string {
	btnText, nextStatus, btnClass := "EMPEZAR", "proceso", "bg-zinc-800 hover:bg-zinc-700 text-zinc-300"
	if estado == "proceso" {
		btnText, nextStatus, btnClass = "TERMINAR", "completado", "bg-blue-600 hover:bg-blue-500 text-white shadow-lg shadow-blue-900/20"
	} else if estado == "completado" {
		btnText, nextStatus, btnClass = "ENTREGAR", "delete", "bg-green-600 hover:bg-green-500 text-white shadow-lg shadow-green-900/20"
	}
	return fmt.Sprintf(`
		<div id="order-%d" class="p-6 md:p-8 bg-zinc-950/80 backdrop-blur-xl border border-zinc-800/80 rounded-[2rem] md:rounded-[2.5rem] shadow-2xl animate-in fade-in zoom-in slide-in-from-top-4 duration-500 group">
			<div class="flex justify-between items-center mb-5">
				<span class="text-[10px] font-black text-zinc-700 uppercase tracking-[0.3em]">#%d</span>
				<span class="px-4 py-1.5 bg-blue-500 text-white text-[10px] font-black uppercase rounded-full tracking-widest shadow-lg shadow-blue-500/20">%s</span>
			</div>
			<div class="mb-8">
				<p class="text-zinc-600 text-[10px] uppercase font-black mb-3 tracking-widest flex items-center gap-2">
                    <span class="w-1 h-1 rounded-full bg-zinc-800"></span> DETALLES DEL PEDIDO
                </p>
				<h3 class="font-bold text-xl md:text-2xl text-zinc-100 leading-tight whitespace-pre-wrap tracking-tight">%s</h3>
			</div>
			<div class="flex">
				<button hx-post="/api/orders/update/%d?status=%s" class="w-full py-5 text-xs font-black uppercase tracking-[0.2em] %s rounded-2xl md:rounded-3xl transition-all active:scale-95">%s</button>
			</div>
		</div>`, id, id, mesa, plato, id, nextStatus, btnClass, btnText)
}
