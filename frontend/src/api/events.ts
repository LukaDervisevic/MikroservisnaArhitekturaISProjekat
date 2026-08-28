import { rpc, asObject } from './http'
import {
  mapEvent,
  mapEventWithLocation,
  mapPage,
  type EventRecord,
  type Page,
} from './types'
import { dateToTimestamp, type WireSpan } from '../lib/codec'

export interface EventInput {
  name: string
  cotisationPrice: number
  agenda: string
  type: string
  /** Sent as {seconds, nanos} — the transcoder rejects RFC3339 strings. */
  dateTime: Date | null
  locationId: number
}

export interface ListEventsParams {
  page: number
  pageSize: number
  type?: string
  fromDate?: Date | null
  toDate?: Date | null
  locationId?: number
}

function eventBody(input: EventInput): Record<string, unknown> {
  const body: Record<string, unknown> = {
    name: input.name,
    cotisation_price: input.cotisationPrice,
    agenda: input.agenda,
    type: input.type,
    location_id: input.locationId,
  }
  // Omit the field entirely when unset; a null would crash the transcoder.
  if (input.dateTime) body.date_time = dateToTimestamp(input.dateTime)
  return body
}

/* Command side — event-service:50052 */

export async function createEvent(input: EventInput): Promise<EventRecord> {
  const raw = await rpc('/api/events', eventBody(input))
  return mapEvent(asObject(raw).event)
}

export async function updateEvent(
  id: number,
  input: EventInput,
): Promise<void> {
  await rpc('/api/events/update', { id, ...eventBody(input) })
}

export async function deleteEvent(id: number): Promise<EventRecord> {
  const raw = await rpc('/api/events/delete', { id })
  return mapEvent(asObject(raw).event)
}

/* Query side — event-query-service:50053 */

/**
 * Note the response key: the proto declares this field as `eventWithLocation`
 * in camelCase, so that — not `event_with_location` — is what comes back.
 */
export async function getEventById(id: number): Promise<EventRecord> {
  const raw = await rpc('/api/events/get-by-id', { id })
  return mapEventWithLocation(asObject(raw).eventWithLocation)
}

export async function getEventByName(name: string): Promise<EventRecord> {
  const raw = await rpc('/api/events/get-by-name', { name })
  return mapEventWithLocation(asObject(raw).eventWithLocation)
}

export async function listEvents(
  params: ListEventsParams,
  signal?: AbortSignal,
): Promise<Page<EventRecord>> {
  const body: Record<string, unknown> = {
    page: params.page,
    page_size: params.pageSize,
    type: params.type ?? '',
    location_id: params.locationId ?? 0,
  }
  let fromDate: WireSpan | undefined
  let toDate: WireSpan | undefined
  if (params.fromDate) fromDate = dateToTimestamp(params.fromDate)
  if (params.toDate) toDate = dateToTimestamp(params.toDate)
  if (fromDate) body.from_date = fromDate
  if (toDate) body.to_date = toDate

  const raw = await rpc('/api/events/list', body, signal)
  return mapPage(raw, 'events', mapEventWithLocation)
}

export async function listEventsByType(
  type: string,
  page: number,
  pageSize: number,
  signal?: AbortSignal,
): Promise<Page<EventRecord>> {
  const raw = await rpc(
    '/api/events/list-by-type',
    { type, page, page_size: pageSize },
    signal,
  )
  return mapPage(raw, 'events', mapEventWithLocation)
}
