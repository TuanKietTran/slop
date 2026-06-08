// Proxy to ingress /api/tasks
export default defineEventHandler(async () => {
  const config = useRuntimeConfig()
  const ingressUrl = config.ingressUrl || 'http://localhost:8080'

  try {
    const data = await $fetch(`${ingressUrl}/api/tasks`)
    return data
  } catch (err) {
    console.error('[tasks] fetch error:', err)
    return []
  }
})
