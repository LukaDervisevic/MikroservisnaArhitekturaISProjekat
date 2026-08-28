/**
 * Domain types plus the mappers that turn decoded gateway payloads into them.
 *
 * The wire uses the proto field names (snake_case, since lua-protobuf decodes
 * to the names as declared) and returns int64 as either a number or a string.
 * Mapping happens once, here, so the rest of the app sees plain camelCase
 * objects with real `Date`s and real `number`s.
 */

import { asArray, asObject } from './http'
import {
  timestampToDate,
  toBoolean,
  toNumber,
  toText,
  type WireSpan,
} from '../lib/codec'

export interface Page<T> {
  items: T[]
  totalCount: number
  page: number
  pageSize: number
  hasNextPage: boolean
}

/** Every list response shares this envelope; only the repeated field differs. */
export function mapPage<T>(
  raw: unknown,
  itemsKey: string,
  mapItem: (item: unknown) => T,
): Page<T> {
  const record = asObject(raw)
  return {
    items: asArray(record[itemsKey]).map(mapItem),
    totalCount: toNumber(record.total_count),
    page: toNumber(record.page),
    pageSize: toNumber(record.page_size),
    hasNextPage: toBoolean(record.has_next_page),
  }
}

/* ------------------------------------------------------------------ */
/* Lecturer                                                            */
/* ------------------------------------------------------------------ */

export interface Lecturer {
  id: number
  fullName: string
  title: string
  fieldOfExpertise: string
  email: string
}

export function mapLecturer(raw: unknown): Lecturer {
  const record = asObject(raw)
  return {
    id: toNumber(record.id),
    fullName: toText(record.full_name),
    title: toText(record.title),
    fieldOfExpertise: toText(record.field_of_expertise),
    email: toText(record.email),
  }
}

/* ------------------------------------------------------------------ */
/* Location                                                            */
/* ------------------------------------------------------------------ */

export interface Location {
  id: number
  name: string
  address: string
  capacity: number
}

export function mapLocation(raw: unknown): Location {
  const record = asObject(raw)
  return {
    id: toNumber(record.id),
    name: toText(record.name),
    address: toText(record.address),
    capacity: toNumber(record.capacity),
  }
}

/* ------------------------------------------------------------------ */
/* Event                                                               */
/* ------------------------------------------------------------------ */

/**
 * The denormalised read-model row returned by every query-side event route.
 * The command side returns a nested `Event` instead — `mapEvent` below
 * flattens it into this same shape so the UI only knows one event type.
 */
export interface EventRecord {
  id: number
  name: string
  cotisationPrice: number
  agenda: string
  type: string
  dateTime: Date | null
  locationId: number
  locationName: string
  locationAddress: string
  locationCapacity: number
}

/** `EventWithLocation`, from event-query-service. */
export function mapEventWithLocation(raw: unknown): EventRecord {
  const record = asObject(raw)
  return {
    id: toNumber(record.event_id),
    name: toText(record.event_name),
    cotisationPrice: toNumber(record.event_cotisation_price),
    agenda: toText(record.event_agenda),
    type: toText(record.event_type),
    dateTime: timestampToDate(record.event_date_time as WireSpan | undefined),
    locationId: toNumber(record.location_id),
    locationName: toText(record.location_name),
    locationAddress: toText(record.location_address),
    locationCapacity: toNumber(record.location_capacity),
  }
}

/** `Event` with a nested `location`, from event-service's command side. */
export function mapEvent(raw: unknown): EventRecord {
  const record = asObject(raw)
  const location = asObject(record.location)
  return {
    id: toNumber(record.id),
    name: toText(record.name),
    cotisationPrice: toNumber(record.cotisation_price),
    agenda: toText(record.agenda),
    type: toText(record.type),
    dateTime: timestampToDate(record.date_time as WireSpan | undefined),
    locationId: toNumber(location.id),
    locationName: toText(location.name),
    locationAddress: toText(location.address),
    locationCapacity: toNumber(location.capacity),
  }
}

/* ------------------------------------------------------------------ */
/* Lecture                                                             */
/* ------------------------------------------------------------------ */

/** The lecture read model, projected from lecture-service over RabbitMQ. */
export interface LectureRecord {
  id: number
  name: string
  durationSeconds: number
  lecturerId: number
  lecturerFullName: string
  lecturerTitle: string
  lecturerFieldOfExpertise: string
  eventId: number
  eventName: string
  cotisationPrice: number
  agenda: string
  eventType: string
  eventDateTime: Date | null
  locationId: number
  locationName: string
  locationAddress: string
  locationCapacity: number
}

export function mapLectureQuery(raw: unknown): LectureRecord {
  const record = asObject(raw)
  return {
    id: toNumber(record.lecture_id),
    name: toText(record.lecture_name),
    durationSeconds: toNumber(
      asObject(record.lecture_duration).seconds,
    ),
    lecturerId: toNumber(record.lecturer_id),
    lecturerFullName: toText(record.lecturer_full_name),
    lecturerTitle: toText(record.lecturer_title),
    lecturerFieldOfExpertise: toText(record.lecturer_field_of_expertise),
    eventId: toNumber(record.event_id),
    eventName: toText(record.event_name),
    cotisationPrice: toNumber(record.event_cotisation_price),
    agenda: toText(record.event_agenda),
    eventType: toText(record.event_type),
    eventDateTime: timestampToDate(
      record.event_date_time as WireSpan | undefined,
    ),
    locationId: toNumber(record.location_id),
    locationName: toText(record.location_name),
    locationAddress: toText(record.location_address),
    locationCapacity: toNumber(record.location_capacity),
  }
}

/**
 * The command side returns a full nested `Lecture`, which — unlike the read
 * model — does carry the lecture's own name and duration.
 */
export interface LectureCommandResult {
  id: number
  name: string
  durationSeconds: number
  lecturerId: number
  lecturerFullName: string
  eventId: number
  eventName: string
}

export function mapLecture(raw: unknown): LectureCommandResult {
  const record = asObject(raw)
  const lecturer = asObject(record.lecturer)
  const event = asObject(record.event)
  const duration = asObject(record.duration)
  return {
    id: toNumber(record.id),
    name: toText(record.name),
    durationSeconds: toNumber(duration.seconds),
    lecturerId: toNumber(lecturer.id),
    lecturerFullName: toText(lecturer.full_name),
    eventId: toNumber(event.id),
    eventName: toText(event.name),
  }
}

/* ------------------------------------------------------------------ */
/* Event sourcing                                                      */
/* ------------------------------------------------------------------ */

export interface SourcedEventMutation {
  aggregateId: string
  version: number
}

export function mapSourcedMutation(raw: unknown): SourcedEventMutation {
  const record = asObject(raw)
  return {
    aggregateId: toText(record.aggregate_id),
    version: toNumber(record.version),
  }
}

export interface SourcedEventState {
  aggregateId: string
  version: number
  name: string
  cotisationPrice: number
  agenda: string
  type: string
  dateTime: Date | null
  locationId: number
  cancelled: boolean
}

export function mapSourcedState(raw: unknown): SourcedEventState {
  const record = asObject(raw)
  return {
    aggregateId: toText(record.aggregate_id),
    version: toNumber(record.version),
    name: toText(record.name),
    cotisationPrice: toNumber(record.cotisation_price),
    agenda: toText(record.agenda),
    type: toText(record.type),
    dateTime: timestampToDate(record.date_time as WireSpan | undefined),
    locationId: toNumber(record.location_id),
    cancelled: toBoolean(record.cancelled),
  }
}

/** One entry in the aggregate's append-only history. */
export interface SourcedEventEntry {
  eventId: string
  aggregateId: string
  version: number
  eventType: string
  occurredAt: Date | null
  /** Opaque JSON blob describing the change; shape varies per event type. */
  details: string
}

export function mapSourcedEntry(raw: unknown): SourcedEventEntry {
  const record = asObject(raw)
  return {
    eventId: toText(record.event_id),
    aggregateId: toText(record.aggregate_id),
    version: toNumber(record.version),
    eventType: toText(record.event_type),
    occurredAt: timestampToDate(record.occurred_at as WireSpan | undefined),
    details: toText(record.details),
  }
}
