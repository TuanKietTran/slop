// SSE proxy: forwards the ingress SSE stream to the browser.
export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig()
  const ingressUrl = config.ingressUrl || 'http://localhost:8080'

  setResponseHeaders(event, {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache',
    'Connection': 'keep-alive',
    'Access-Control-Allow-Origin': '*',
  })

  // Proxy the SSE stream from ingress.
  const controller = new AbortController()
  event.node.req.on('close', () => controller.abort())

  try {
    const upstream = await fetch(`${ingressUrl}/api/stream`, {
      signal: controller.signal,
      headers: { Accept: 'text/event-stream' },
    })

    if (!upstream.body) {
      await sendStream(event, new ReadableStream({
        start(c) { c.close() },
      }))
      return
    }

    await sendStream(event, upstream.body)
  } catch (err: any) {
    if (err?.name !== 'AbortError') {
      console.error('[stream] upstream error:', err)
    }
  }
})
