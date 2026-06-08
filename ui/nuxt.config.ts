// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  devtools: { enabled: true },
  modules: ['@nuxtjs/tailwindcss'],
  css: ['@vue-flow/core/dist/style.css', '@vue-flow/core/dist/theme-default.css'],
  runtimeConfig: {
    ingressUrl: process.env.INGRESS_URL || 'http://localhost:8080',
    public: {
      ingressUrl: process.env.NUXT_PUBLIC_INGRESS_URL || 'http://localhost:8080',
    },
  },
  nitro: {
    routeRules: {
      '/api/stream': { headers: { 'Content-Type': 'text/event-stream' } },
    },
  },
  compatibilityDate: '2024-04-03',
})
