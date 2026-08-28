/**
 * Wire codecs for APISIX's `grpc-transcode` plugin.
 *
 * APISIX 3.14.1 does NOT implement proto3's canonical JSON mapping for the
 * well-known types. It treats every message-typed field as a plain nested
 * object, so `google.protobuf.Timestamp` and `google.protobuf.Duration` must
 * travel as `{seconds, nanos}` rather than as RFC3339 / duration strings.
 * Sending a string there crashes the transcoder with a 500
 * ("bad argument #1 to 'isarray'").
 *
 * Every conversion between JS values and the wire lives in this file so the
 * quirk never leaks into components. See api-gateway/README.md.
 */

/** `google.protobuf.Timestamp` / `google.protobuf.Duration` on the wire. */
export interface WireSpan {
  seconds?: number | string
  nanos?: number | string
}

/**
 * int64 fields may decode as either a number or a string depending on
 * magnitude, so every numeric field from the wire goes through here.
 */
export function toNumber(value: unknown): number {
  if (typeof value === 'number') return value
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    if (!Number.isNaN(parsed)) return parsed
  }
  return 0
}

export function toText(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

export function toBoolean(value: unknown): boolean {
  return value === true
}

/** Wire timestamp -> Date. Returns null for an absent or zero timestamp. */
export function timestampToDate(span: WireSpan | null | undefined): Date | null {
  if (!span) return null
  const seconds = toNumber(span.seconds)
  const nanos = toNumber(span.nanos)
  if (seconds === 0 && nanos === 0) return null
  return new Date(seconds * 1000 + Math.floor(nanos / 1e6))
}

/** Date -> wire timestamp. */
export function dateToTimestamp(date: Date): WireSpan {
  const millis = date.getTime()
  return {
    seconds: Math.floor(millis / 1000),
    nanos: (millis % 1000) * 1e6,
  }
}

/**
 * Value of a `<input type="datetime-local">` -> wire timestamp.
 * The input yields wall-clock time with no zone, which `new Date()` reads as
 * local time — the same convention the form displays, so it round-trips.
 */
export function datetimeLocalToTimestamp(value: string): WireSpan | undefined {
  if (!value) return undefined
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return undefined
  return dateToTimestamp(date)
}

/** Date -> `<input type="datetime-local">` value, in local time. */
export function dateToDatetimeLocal(date: Date | null): string {
  if (!date) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return (
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}` +
    `T${pad(date.getHours())}:${pad(date.getMinutes())}`
  )
}

/** Wire duration -> whole minutes. */
export function durationToMinutes(span: WireSpan | null | undefined): number {
  if (!span) return 0
  return Math.round(toNumber(span.seconds) / 60)
}

/** Whole minutes -> wire duration. */
export function minutesToDuration(minutes: number): WireSpan {
  return { seconds: Math.round(minutes) * 60, nanos: 0 }
}
