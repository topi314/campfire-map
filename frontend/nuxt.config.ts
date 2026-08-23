export default defineNuxtConfig({
  ssr: false,
  compatibilityDate: "2025-01-01",
  css: [
    "leaflet/dist/leaflet.css",
    "leaflet.markercluster/dist/MarkerCluster.css",
    "leaflet.markercluster/dist/MarkerCluster.Default.css",
    "~/assets/css/main.css",
  ],
  runtimeConfig: {
    public: {
      // Empty in production (same-origin via Go). Set in dev via env or leave blank and use Vite proxy.
      apiBase: process.env.NUXT_PUBLIC_API_BASE || "",
    },
  },
  nitro: {
    preset: "static",
  },
  routeRules: {
    "/api/**": { proxy: "http://127.0.0.1:8080/api/**" },
  },
  app: {
    head: {
      title: "Pokémon GO map",
      meta: [{ name: "viewport", content: "width=device-width, initial-scale=1" }],
    },
  },
  vite: {
    server: {
      proxy: {
        "/api": { target: "http://127.0.0.1:8080", changeOrigin: true },
      },
    },
    optimizeDeps: {
      include: ["leaflet", "leaflet.markercluster"],
    },
  },
});
