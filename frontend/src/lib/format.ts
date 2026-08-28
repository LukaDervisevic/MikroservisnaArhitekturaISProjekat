const dateTimeFormat = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short',
})

export function formatDateTime(date: Date | null): string {
  return date ? dateTimeFormat.format(date) : '—'
}

const currencyFormat = new Intl.NumberFormat(undefined, {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

export function formatPrice(value: number): string {
  return currencyFormat.format(value)
}

export function formatDuration(totalSeconds: number): string {
  if (!totalSeconds) return '—'
  const minutes = Math.round(totalSeconds / 60)
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  if (!hours) return `${rest} min`
  return rest ? `${hours} h ${rest} min` : `${hours} h`
}

/** Pretty-prints the opaque `details` blob on a sourced event, if it is JSON. */
export function formatDetails(details: string): string {
  if (!details) return ''
  try {
    return JSON.stringify(JSON.parse(details), null, 2)
  } catch {
    return details
  }
}
