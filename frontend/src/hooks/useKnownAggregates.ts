/**
 * Remembers event ids opened on the Event Sourcing page in this browser.
 *
 * Since event-service is unified, any event id works here (including ones
 * created from the Events page — the "History" action there deep-links in).
 * The event-sourcing routes have no "list" of their own, so this is just a
 * local shortcut list; it is not a source of truth and never leaves the browser.
 */

import { useCallback, useState } from 'react'

const STORAGE_KEY = 'mais.knownAggregates'
const LIMIT = 25

export interface KnownAggregate {
  id: string
  label: string
}

function read(): KnownAggregate[] {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (entry): entry is KnownAggregate =>
        typeof entry === 'object' &&
        entry !== null &&
        typeof (entry as KnownAggregate).id === 'string',
    )
  } catch {
    // Private browsing, cleared site data, or storage disabled entirely.
    return []
  }
}

function write(entries: KnownAggregate[]) {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(entries))
  } catch {
    // Non-fatal: the page still works, it just won't remember next reload.
  }
}

export function useKnownAggregates() {
  const [aggregates, setAggregates] = useState<KnownAggregate[]>(read)

  const remember = useCallback((id: string, label: string) => {
    if (!id) return
    setAggregates((current) => {
      const next = [
        { id, label },
        ...current.filter((entry) => entry.id !== id),
      ].slice(0, LIMIT)
      write(next)
      return next
    })
  }, [])

  const forget = useCallback((id: string) => {
    setAggregates((current) => {
      const next = current.filter((entry) => entry.id !== id)
      write(next)
      return next
    })
  }, [])

  return { aggregates, remember, forget }
}
