package frontend

import (
	"html/template"
	"net/http"
)

var indexTmpl = template.Must(template.New("index").Parse(`
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
            Lógica en <span class="text-blue-400">/backend</span>, Interfaz en <span class="text-blue-400">/frontend</span>. 
            Separación física total.
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

// HandleIndex es pública ahora (mayúscula)
func HandleIndex(w http.ResponseWriter, r *http.Request) {
	indexTmpl.Execute(w, nil)
}
