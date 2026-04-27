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
        <div class="max-w-7xl mx-auto flex items-center justify-between">
            <div class="flex gap-4 md:gap-8">
                <a href="/" class="flex items-center gap-2 hover:text-blue-400 font-black uppercase text-[10px] md:text-xs tracking-[0.2em] transition-all">
                    <span class="w-2 h-2 rounded-full bg-blue-500"></span>Camareros
                </a>
                <a href="/cocina" class="flex items-center gap-2 hover:text-green-400 font-black uppercase text-[10px] md:text-xs tracking-[0.2em] transition-all">
                    <span class="w-2 h-2 rounded-full bg-green-500"></span>Cocina
                </a>
            </div>
            <a href="/config" class="text-zinc-500 hover:text-white transition-all p-2 hover:bg-zinc-800 rounded-2xl border border-transparent hover:border-zinc-700">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/>
                </svg>
            </a>
        </div>
    </nav>
`

const layoutFooter = `
    <footer class="mt-20 p-8 border-t border-zinc-900 text-center">
        <p class="text-[10px] text-zinc-600 font-black uppercase tracking-[0.3em]">Comandas</p>
    </footer>
</body>
</html>
`

// --- VISTA COMANDAS ---
var comandasTmpl = template.Must(template.New("comandas").Parse(layoutHeader + `
    <main class="max-w-2xl mx-auto p-4 md:p-8 relative">
        <div class="absolute top-40 left-[-20%] -inset-10 bg-blue-500/10 blur-[120px] rounded-full w-64 md:w-96 h-64 md:h-96 opacity-30"></div>
        <div class="relative z-10 mt-4 md:mt-10">
            <header class="mb-8">
                <h2 class="text-3xl md:text-5xl font-black mb-2 tracking-tighter">Nueva Comanda</h2>
                <p class="text-zinc-500 text-[10px] md:text-xs uppercase font-bold tracking-[0.2em]">Panel de Camareros</p>
            </header>
            <div class="mb-8 relative">
                <label class="block text-[10px] text-blue-400 uppercase font-black mb-3 ml-1 tracking-widest">Buscador Rápido</label>
                <div class="relative group">
                    <input type="text" id="product-search" name="q" hx-get="/api/products/search" hx-trigger="keyup changed delay:300ms" hx-target="#search-results" placeholder="Escribe para buscar..." class="w-full bg-zinc-900/40 backdrop-blur-xl border border-zinc-800 p-4 md:p-6 rounded-2xl md:rounded-3xl focus:ring-2 focus:ring-blue-500 outline-none transition-all placeholder:text-zinc-700 text-lg">
                    <div class="absolute right-4 top-1/2 -translate-y-1/2 text-zinc-700 group-focus-within:text-blue-500 transition-colors">
                        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
                    </div>
                </div>
                <div id="search-results" class="absolute w-full mt-3 z-[60] flex flex-col gap-2 drop-shadow-2xl"></div>
            </div>
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
            </form>
        </div>
    </main>
` + layoutFooter))

// --- VISTA CONFIGURACIÓN ---
var configTmpl = template.Must(template.New("config").Parse(layoutHeader + `
    <main class="max-w-6xl mx-auto p-4 md:p-8 relative">
        <div class="absolute top-20 left-[-10%] bg-blue-500/5 blur-[120px] rounded-full w-96 h-96 opacity-30"></div>
        <header class="mb-12">
            <h2 class="text-4xl md:text-6xl font-black mb-2 tracking-tighter uppercase">Gestión del Menú</h2>
        </header>
        <div class="grid grid-cols-1 lg:grid-cols-12 gap-8 md:gap-12 relative z-10">
            <div class="lg:col-span-5">
                <h3 class="text-zinc-500 uppercase text-[10px] font-black tracking-widest mb-6 ml-2">Añadir Nuevo Producto</h3>
                <form hx-post="/api/products" hx-target="#product-list" hx-on::after-request="this.reset()" class="flex flex-col gap-5 bg-zinc-900/40 p-6 md:p-8 rounded-[2.5rem] border border-zinc-800/50 backdrop-blur-md">
                    <input type="text" name="name" oninput="formatName(this)" placeholder="Nombre del Producto" class="bg-zinc-950/50 border border-zinc-800 p-4 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500 transition-all" required>
                    <input type="text" name="price" oninput="formatPrice(this)" placeholder="Precio (CLP)" class="bg-zinc-950/50 border border-zinc-800 p-4 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500 transition-all" required>
                    <textarea name="description" oninput="formatSentence(this)" placeholder="Descripción breve..." class="bg-zinc-950/50 border border-zinc-800 p-4 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500 h-28 transition-all" required></textarea>
                    <button type="submit" class="bg-blue-600 hover:bg-blue-500 py-5 rounded-2xl font-black uppercase text-xs tracking-widest shadow-xl shadow-blue-900/20 active:scale-95 transition-all">Guardar en el Menú</button>
                </form>
            </div>
            <div class="lg:col-span-7">
                <h3 class="text-zinc-500 uppercase text-[10px] font-black tracking-widest mb-6 ml-2">Productos Registrados</h3>
                <div id="product-list" hx-get="/api/products" hx-trigger="load" class="grid grid-cols-1 gap-4 max-h-[70vh] overflow-y-auto pr-2 custom-scrollbar">
                    <div class="p-8 text-center border-2 border-dashed border-zinc-900 rounded-[2rem] text-zinc-700 font-bold uppercase text-[10px] tracking-widest">Cargando inventario...</div>
                </div>
            </div>
        </div>
    </main>
` + layoutFooter))

func handleComandas(w http.ResponseWriter, r *http.Request) { comandasTmpl.Execute(w, nil) }
func handleConfig(w http.ResponseWriter, r *http.Request) { configTmpl.Execute(w, nil) }

func RenderCocinaPage(w http.ResponseWriter, orders []Order) {
	w.Write([]byte(layoutHeader))
	w.Write([]byte(`
    <main class="p-4 md:p-8 relative" hx-ext="ws" ws-connect="/ws">
        <div class="absolute top-40 left-1/4 -inset-10 bg-blue-500/5 blur-3xl rounded-full w-96 h-96 opacity-20"></div>
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-6 md:gap-8 relative z-10">
            <div class="flex flex-col gap-5">
                <div class="flex items-center gap-3 px-4"><div class="w-2 h-2 rounded-full bg-yellow-500 animate-pulse"></div><h2 class="font-black text-zinc-400 text-[10px] uppercase tracking-[0.2em]">Pendientes</h2></div>
                <div id="pedidos-col" class="kanban-col flex flex-col gap-4 p-3 md:p-5 bg-zinc-900/20 backdrop-blur-md border border-zinc-800/50 rounded-[2.5rem] md:rounded-[3rem]">`))
	for _, o := range orders { if o.Estado == "pendiente" { w.Write([]byte(RenderOrderCard(o.ID, o.Mesa, o.Plato, o.Estado))) } }
	w.Write([]byte(`</div>
            </div>
            <div class="flex flex-col gap-5">
                <div class="flex items-center gap-3 px-4"><div class="w-2 h-2 rounded-full bg-blue-500 animate-pulse"></div><h2 class="font-black text-zinc-400 text-[10px] uppercase tracking-[0.2em]">En Proceso</h2></div>
                <div id="proceso-col" class="kanban-col flex flex-col gap-4 p-3 md:p-5 bg-zinc-900/20 backdrop-blur-md border border-zinc-800/50 rounded-[2.5rem] md:rounded-[3rem]">`))
	for _, o := range orders { if o.Estado == "proceso" { w.Write([]byte(RenderOrderCard(o.ID, o.Mesa, o.Plato, o.Estado))) } }
	w.Write([]byte(`</div>
            </div>
            <div class="flex flex-col gap-5">
                <div class="flex items-center gap-3 px-4"><div class="w-2 h-2 rounded-full bg-green-500 animate-pulse"></div><h2 class="font-black text-zinc-400 text-[10px] uppercase tracking-[0.2em]">Completado</h2></div>
                <div id="completado-col" class="kanban-col flex flex-col gap-4 p-3 md:p-5 bg-zinc-900/20 backdrop-blur-md border border-zinc-800/50 rounded-[2.5rem] md:rounded-[3rem]">`))
	for _, o := range orders { if o.Estado == "completado" { w.Write([]byte(RenderOrderCard(o.ID, o.Mesa, o.Plato, o.Estado))) } }
	w.Write([]byte(`</div>
            </div>
        </div>
    </main>
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
