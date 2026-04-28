package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
)

// -------------------------------------
// -------------| Helpers
// -------------------------------------
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

// -------------------------------------
// -------------| Plantillas
// -------------------------------------
const layoutHeader = `
<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>Comandas App</title>
    <meta name="view-transition" content="same-origin">
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;700;900&display=swap" rel="stylesheet">
    <style> body { font-family: 'Inter', sans-serif; } </style>
    <script src="https://unpkg.com/htmx.org@1.9.11"></script>
    <script src="https://unpkg.com/htmx.org/dist/ext/ws.js"></script>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://instant.page/5.2.0" type="module" integrity="sha384-jnZyxPjiipSbm6WFEJrqQUqzHKyeWMLzQoUf69GON5m929OdzP64VGupxzGTzG++"></script>
    <script>
        // Registro de Service Worker
        if ('serviceWorker' in navigator) {
            window.addEventListener('load', () => {
                navigator.serviceWorker.register('/sw.js');
            });
        }
        // Habilitar transiciones globales en HTMX
        document.addEventListener("DOMContentLoaded", () => {
            htmx.config.globalViewTransitions = true;
        });

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

        /* Animación suave entre páginas */
        ::view-transition-old(root),
        ::view-transition-new(root) {
            animation-duration: 0.3s;
        }
    </style>
</head>
<body class="bg-zinc-950 text-zinc-50 antialiased overflow-x-hidden">
`

const layoutFooter = `
</body>
</html>
`

// -------------------------------------
// -------------| Header
// -------------------------------------
func RenderHeader(title string) string {
	return fmt.Sprintf(`
        <header class="flex items-center gap-4 px-6 md:px-12 py-8 bg-[#09090b]">
            <div class="w-10 h-10 bg-blue-600 rounded-xl flex items-center justify-center shadow-lg shadow-blue-600/20">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 12h6"/><path d="M9 16h6"/><path d="M9 8h6"/></svg>
            </div>
            <h1 class="text-white font-black tracking-[0.2em] text-xs md:text-sm uppercase">%s</h1>
        </header>`, title)
}

// -------------------------------------
// -------------| Bottom Bar
// -------------------------------------
func RenderBottomBar(active string) string {
	getClass := func(tab string) string {
		base := "text-[10px] font-black uppercase tracking-[0.3em] transition-all px-8 py-3 rounded-full border border-transparent"
		if tab == active {
			return base + " bg-blue-600 text-white shadow-2xl shadow-blue-600/40"
		}
		return base + " text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900/50"
	}
	return fmt.Sprintf(`
        <nav class="fixed bottom-0 left-0 w-full bg-[#09090b] py-8 px-6 flex items-center justify-center z-[100]">
            <div class="flex items-center gap-4 md:gap-8">
                <a href="/" class="%s">Comandas</a>
                <a href="/cocina" class="%s">Cocina</a>
                <a href="/config" class="%s">Ajustes</a>
            </div>
        </nav>`, getClass("comandas"), getClass("cocina"), getClass("config"))
}

// -------------------------------------
// -------------| Comandas
// -------------------------------------
var comandasTmpl = template.Must(template.New("comandas").Parse(`
    <!-- VISTA 1: SELECCIÓN DE MESA -->
    <div id="view-tables" class="relative z-10">
        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-6">
            {{range .Tables}}
            <button type="button" onclick="selectTable('{{.Name}}', this)" class="group aspect-square bg-zinc-900/40 border border-zinc-800/50 rounded-[2.5rem] flex items-center justify-center transition-all hover:bg-zinc-800 hover:border-zinc-700 active:scale-95">
                <span class="font-black text-xl md:text-2xl tracking-tight text-white">{{.Name}}</span>
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
    <div id="view-order" class="hidden relative z-10">
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
`))

func RenderComandasPage(w http.ResponseWriter, tables []Table) {
	w.Write([]byte(layoutHeader))
	w.Write([]byte(RenderHeader("Seleccionar Mesa")))
	w.Write([]byte(`<main class="max-w-7xl mx-auto px-6 md:px-12 pb-32">`))
	comandasTmpl.Execute(w, map[string]interface{}{"Tables": tables})
	w.Write([]byte(`</main>`))
	w.Write([]byte(RenderBottomBar("comandas")))
	w.Write([]byte(layoutFooter))
}

func handleComandas(w http.ResponseWriter, _ *http.Request) { RenderComandasPage(w, nil) }

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

// -------------------------------------
// -------------| Cocina
// -------------------------------------
func RenderCocinaPage(w http.ResponseWriter, orders []Order) {
	counts := map[string]int{"pendiente": 0, "proceso": 0, "completado": 0}
	for _, o := range orders {
		counts[o.Estado]++
	}

	w.Write([]byte(layoutHeader))
	w.Write([]byte(RenderHeader("Panel de Cocina")))
	w.Write([]byte(`<main class="max-w-7xl mx-auto px-6 md:px-12 pb-32" hx-ext="ws" ws-connect="/ws">
            <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
                <!-- PENDIENTE -->
                <div class="flex flex-col gap-4 bg-zinc-900/20 border border-zinc-800/50 rounded-[2.5rem] p-4 min-h-[70vh]">
                    <div class="flex items-center justify-between px-4 py-2">
                        <h2 class="font-black text-[10px] uppercase tracking-[0.2em] text-zinc-500">Pendiente</h2>
                        <span class="w-5 h-5 flex items-center justify-center bg-zinc-800/50 rounded-md text-[9px] font-bold text-zinc-500">` + strconv.Itoa(counts["pendiente"]) + `</span>
                    </div>
                    <div id="pedidos-col" class="flex flex-col gap-4">`))
	for _, o := range orders {
		if o.Estado == "pendiente" {
			w.Write([]byte(RenderOrderCard(o.ID, o.Mesa, o.Plato, o.Estado)))
		}
	}
	w.Write([]byte(`</div>
                </div>

                <!-- EN PROCESO -->
                <div class="flex flex-col gap-4 bg-zinc-900/20 border border-zinc-800/50 rounded-[2.5rem] p-4 min-h-[70vh]">
                    <div class="flex items-center justify-between px-4 py-2">
                        <h2 class="font-black text-[10px] uppercase tracking-[0.2em] text-blue-500">En Proceso</h2>
                        <span class="w-5 h-5 flex items-center justify-center bg-blue-500/10 rounded-md text-[9px] font-bold text-blue-500">` + strconv.Itoa(counts["proceso"]) + `</span>
                    </div>
                    <div id="proceso-col" class="flex flex-col gap-4">`))
	for _, o := range orders {
		if o.Estado == "proceso" {
			w.Write([]byte(RenderOrderCard(o.ID, o.Mesa, o.Plato, o.Estado)))
		}
	}
	w.Write([]byte(`</div>
                </div>

                <!-- COMPLETADO -->
                <div class="flex flex-col gap-4 bg-zinc-900/20 border border-zinc-800/50 rounded-[2.5rem] p-4 min-h-[70vh]">
                    <div class="flex items-center justify-between px-4 py-2">
                        <h2 class="font-black text-[10px] uppercase tracking-[0.2em] text-green-500">Completado</h2>
                        <span class="w-5 h-5 flex items-center justify-center bg-green-500/10 rounded-md text-[9px] font-bold text-green-500">` + strconv.Itoa(counts["completado"]) + `</span>
                    </div>
                    <div id="completado-col" class="flex flex-col gap-4">`))
	for _, o := range orders {
		if o.Estado == "completado" {
			w.Write([]byte(RenderOrderCard(o.ID, o.Mesa, o.Plato, o.Estado)))
		}
	}
	w.Write([]byte(`</div>
                </div>
            </div>
        </main>`))
	w.Write([]byte(RenderBottomBar("cocina")))
	w.Write([]byte(layoutFooter))
}

func RenderOrderCard(id int, mesa, plato, estado string) string {
	btnText, nextStatus, btnClass := "EMPEZAR", "proceso", "bg-zinc-800/50 hover:bg-zinc-700 text-zinc-400"
	if estado == "proceso" {
		btnText, nextStatus, btnClass = "TERMINAR", "completado", "bg-blue-600 hover:bg-blue-500 text-white shadow-lg shadow-blue-600/20"
	} else if estado == "completado" {
		btnText, nextStatus, btnClass = "ENTREGAR", "delete", "bg-green-600 hover:bg-green-500 text-white shadow-lg shadow-green-600/20"
	}

	return fmt.Sprintf(`
		<div id="order-%d" class="p-6 bg-zinc-900/60 border border-zinc-800/50 rounded-[2rem] flex flex-col gap-4 animate-in fade-in zoom-in duration-300">
			<div class="flex flex-col gap-1">
				<span class="text-[8px] font-black text-zinc-600 uppercase tracking-[0.2em]">%s</span>
				<h3 class="font-bold text-lg text-white leading-tight tracking-tight">%s</h3>
			</div>
			<button hx-post="/api/orders/update/%d?status=%s" class="w-full py-4 text-[10px] font-black uppercase tracking-[0.2em] %s rounded-2xl transition-all active:scale-95">%s</button>
		</div>`, id, mesa, plato, id, nextStatus, btnClass, btnText)
}

// -------------------------------------
// -------------| Ajustes
// -------------------------------------
var configTmpl = template.Must(template.New("config").Parse(`
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
`))

func RenderConfigPage(w http.ResponseWriter, tables []Table) {
	w.Write([]byte(layoutHeader))
	w.Write([]byte(RenderHeader("Configuraciones")))
	w.Write([]byte(`<main class="max-w-7xl mx-auto px-6 md:px-12 pb-32 relative">`))
	configTmpl.Execute(w, map[string]interface{}{"Tables": tables})
	w.Write([]byte(`</main>`))
	w.Write([]byte(RenderBottomBar("config")))
	w.Write([]byte(layoutFooter))
}

func handleConfig(w http.ResponseWriter, _ *http.Request) { RenderConfigPage(w, nil) }

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
