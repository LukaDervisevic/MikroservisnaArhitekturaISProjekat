/**
 * Remembers aggregate IDs seen in this browser.
 *
 * event-service exposes no "list aggregates" route — every event-sourcing
 * route addresses one aggregate by ID — so without this, an ID would have to
 * be copied by hand after every create. Purely a local convenience; it is not
 * a source of truth and never leaves the browser.
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
