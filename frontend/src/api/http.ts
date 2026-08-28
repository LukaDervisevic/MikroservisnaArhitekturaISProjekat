/**
 * Transport for the APISIX gateway.
 *
 * Every backend route is a POST whose JSON body maps 1:1 onto the proto
 * request message — `grpc-transcode` does not lift path or query params into
 * fields, because these protos carry no `google.api.http` annotations. So
 * there is exactly one call shape, and it lives here.
 *
 * Requests go to same-origin `/api/*`; Vite proxies them to the gateway
 * (see vite.config.ts). The gateway has no `cors` plugin, so calling
 * :9080 directly from the browser would fail at preflight.
 */

export class ApiError extends Error {
  readonly status: number
  readonly grpcMessage: string | null

  constructor(message: string, status: number, grpcMessage: string | null) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.grpcMessage = grpcMessage
  }

  /** A 429 from the gateway's global limit-count rule (100 req / 60s per IP). */
  get isRateLimited(): boolean {
    return this.status === 429
  }
}

type JsonObject = Record<string, unknown>

async function readBody(response: Response): Promise<unknown> {
  const text = await response.text()
  if (!text) return null
  try {
    return JSON.parse(text) as unknown
  } catch {
    return text
  }
}

function extractMessage(body: unknown): string | null {
  if (typeof body === 'string') return body.trim() || null
  if (body && typeof body === 'object') {
    const record = body as JsonObject
    for (const key of ['error_msg', 'message', 'error']) {
      const value = record[key]
      if (typeof value === 'string' && value) return value
    }
  }
  return null
}

/**
 * Calls one gateway route. `TWire` is the raw decoded response; callers are
 * expected to run it through a mapper rather than hand it to components.
 */
export async function rpc<TWire>(
  path: string,
  body: JsonObject = {},
  signal?: AbortSignal,
): Promise<TWire> {
  let response: Response
  try {
    response = await fetch(path, {
      method: 'POST',
      // Required. Without it grpc-transcode skips JSON parsing entirely and
      // every field silently arrives empty.
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      signal,
    })
  } catch (cause) {
    if (signal?.aborted) throw cause
    throw new ApiError(
      'Could not reach the API gateway. Is `docker compose up` running?',
      0,
      null,
    )
  }

  const payload = await readBody(response)

  if (!response.ok) {
    // grpc-transcode surfaces the gRPC status message in this header, which
    // is usually far more specific than the JSON body.
    const grpcMessage = response.headers.get('grpc-message')
    const detail =
      grpcMessage ??
      extractMessage(payload) ??
      `Request failed with status ${response.status}`

    if (response.status === 429) {
      throw new ApiError(
        'Rate limit reached (100 requests/minute). Wait a moment and retry.',
        429,
        grpcMessage,
      )
    }
    throw new ApiError(detail, response.status, grpcMessage)
  }

  // Methods returning google.protobuf.Empty come back as `{}`.
  return (payload ?? {}) as TWire
}

/** Narrows an unknown decoded value to an object for mapping. */
export function asObject(value: unknown): JsonObject {
  return value && typeof value === 'object' ? (value as JsonObject) : {}
}

/** Narrows a repeated field, which is absent rather than `[]` when empty. */
export function asArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : []
}
