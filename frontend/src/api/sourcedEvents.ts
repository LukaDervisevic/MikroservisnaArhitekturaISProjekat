/**
 * The granular, one-field-per-call API for an event. event-service is unified
 * onto a single event-sourced aggregate, so these routes and the coarse
 * `/api/events/*` routes mutate the SAME event: every call appends a domain
 * event, runs the orchestrated saga, and propagates to the read stores.
 *
 * `aggregate_id` is the event's integer id (the one `/api/events/get-by-id`
 * uses), passed as a string because the proto field is `string`. Every mutation
 * returns the new version rather than the new state; call `getSourcedEventState`
 * to read the folded aggregate.
 */

import { rpc, asObject, asArray } from './http'
import {
  mapSourcedEntry,
  mapSourcedMutation,
  mapSourcedState,
  type SourcedEventEntry,
  type SourcedEventMutation,
  type SourcedEventState,
} from './types'
import { dateToTimestamp } from '../lib/codec'

export interface CreateSourcedEventInput {
  name: string
  cotisationPrice: number
  agenda: string
  type: string
  dateTime: Date | null
  locationId: number
}

export async function createSourcedEvent(
  input: CreateSourcedEventInput,
): Promise<SourcedEventMutation> {
  const body: Record<string, unknown> = {
    name: input.name,
    cotisation_price: input.cotisationPrice,
    agenda: input.agenda,
    type: input.type,
    location_id: input.locationId,
  }
  if (input.dateTime) body.date_time = dateToTimestamp(input.dateTime)
  return mapSourcedMutation(await rpc('/api/sourced-events', body))
}

export async function renameSourcedEvent(
  aggregateId: string,
  newName: string,
): Promise<SourcedEventMutation> {
  const raw = await rpc('/api/sourced-events/rename', {
    aggregate_id: aggregateId,
    new_name: newName,
  })
  return mapSourcedMutation(raw)
}

export async function rescheduleSourcedEvent(
  aggregateId: string,
  newDateTime: Date,
): Promise<SourcedEventMutation> {
  const raw = await rpc('/api/sourced-events/reschedule', {
    aggregate_id: aggregateId,
    new_date_time: dateToTimestamp(newDateTime),
  })
  return mapSourcedMutation(raw)
}

export async function relocateSourcedEvent(
  aggregateId: string,
  newLocationId: number,
): Promise<SourcedEventMutation> {
  const raw = await rpc('/api/sourced-events/relocate', {
    aggregate_id: aggregateId,
    new_location_id: newLocationId,
  })
  return mapSourcedMutation(raw)
}

export async function changeSourcedEventPrice(
  aggregateId: string,
  newPrice: number,
): Promise<SourcedEventMutation> {
  const raw = await rpc('/api/sourced-events/change-price', {
    aggregate_id: aggregateId,
    new_price: newPrice,
  })
  return mapSourcedMutation(raw)
}

export async function cancelSourcedEvent(
  aggregateId: string,
  reason: string,
): Promise<SourcedEventMutation> {
  const raw = await rpc('/api/sourced-events/cancel', {
    aggregate_id: aggregateId,
    reason,
  })
  return mapSourcedMutation(raw)
}

/** Replays the aggregate (from its latest snapshot) to its current state. */
export async function getSourcedEventState(
  aggregateId: string,
  signal?: AbortSignal,
): Promise<SourcedEventState> {
  const raw = await rpc(
    '/api/sourced-events/get-state',
    { aggregate_id: aggregateId },
    signal,
  )
  return mapSourcedState(raw)
}

export async function getSourcedEventHistory(
  aggregateId: string,
  signal?: AbortSignal,
): Promise<SourcedEventEntry[]> {
  const raw = await rpc(
    '/api/sourced-events/get-history',
    { aggregate_id: aggregateId },
    signal,
  )
  return asArray(asObject(raw).events).map(mapSourcedEntry)
}

export async function createSourcedEventSnapshot(
  aggregateId: string,
): Promise<SourcedEventMutation> {
  const raw = await rpc('/api/sourced-events/create-snapshot', {
    aggregate_id: aggregateId,
  })
  return mapSourcedMutation(raw)
}
