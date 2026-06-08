// Proxy to ingress /api/reviews/ready
export default defineEventHandler(async () => {
  const config = useRuntimeConfig()
  const ingressUrl = config.ingressUrl || 'http://localhost:8080'

  try {
    const data = await $fetch(`${ingressUrl}/api/reviews/ready`)
    return data
  } catch (err) {
    console.error('[reviews/ready] fetch error:', err)
    return []
  }
})
