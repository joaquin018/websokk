import { component$ } from "@builder.io/qwik";
import type { DocumentHead } from "@builder.io/qwik-city";

export default component$(() => {
  return (
    <main class="flex min-h-screen flex-col items-center justify-center p-6 text-center">
      <div class="relative flex flex-col items-center gap-6">
        <div class="absolute -inset-10 bg-blue-500/20 blur-3xl rounded-full"></div>
        
        <h1 class="text-6xl font-bold tracking-tighter sm:text-7xl bg-linear-to-b from-white to-zinc-500 bg-clip-text text-transparent">
          Comandas App
        </h1>
        
        <p class="max-w-md text-zinc-400 text-lg sm:text-xl">
          El sistema de gestión de pedidos más rápido del mundo. 
          Desarrollado con <span class="text-blue-400">Qwik</span>, <span class="text-blue-400">Go</span> y <span class="text-blue-400">PostgreSQL</span>.
        </p>

        <div class="flex gap-4 mt-4">
          <button class="px-8 py-3 bg-white text-black font-semibold rounded-full hover:bg-zinc-200 transition-colors">
            Ver Mesas
          </button>
          <button class="px-8 py-3 border border-zinc-800 rounded-full hover:bg-zinc-900 transition-colors">
            Configuración
          </button>
        </div>
      </div>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Comandas App | Máxima Velocidad",
  meta: [
    {
      name: "description",
      content: "Aplicación de comandas ultrarrápida",
    },
  ],
};
