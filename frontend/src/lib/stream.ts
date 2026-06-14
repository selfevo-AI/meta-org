export interface StreamEvent<T = unknown> {
  event: string
  data: T
}

function parseFrame<T>(frame: string): StreamEvent<T> | null {
  const lines = frame.split(/\r?\n/)
  const eventLine = lines.find((line) => line.startsWith('event:'))
  const dataLines = lines.filter((line) => line.startsWith('data:'))

  if (dataLines.length === 0) return null

  const data = dataLines.map((line) => line.replace(/^data:\s?/, '')).join('\n')
  return {
    event: eventLine ? eventLine.replace(/^event:\s?/, '').trim() : 'message',
    data: JSON.parse(data) as T,
  }
}

export async function streamSSE<T>(
  url: string,
  token: string,
  onEvent: (event: StreamEvent<T>) => void,
  signal?: AbortSignal,
) {
  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
    signal,
  })
  if (!response.ok || !response.body) {
    throw new Error(`HTTP ${response.status}`)
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const frames = buffer.split(/\r?\n\r?\n/)
    buffer = frames.pop() ?? ''
    for (const frame of frames) {
      const parsed = parseFrame<T>(frame)
      if (parsed) onEvent(parsed)
    }
  }

  const finalFrame = buffer.trim()
  if (finalFrame) {
    const parsed = parseFrame<T>(finalFrame)
    if (parsed) onEvent(parsed)
  }
}
