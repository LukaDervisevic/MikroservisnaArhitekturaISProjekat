import { rpc, asObject } from './http'
import {
  mapLecture,
  mapLectureQuery,
  mapPage,
  type LectureCommandResult,
  type LectureRecord,
  type Page,
} from './types'
import { minutesToDuration } from '../lib/codec'

export interface LectureInput {
  lecturerId: number
  eventId: number
  name: string
  /** Sent as a {seconds, nanos} Duration. */
  durationMinutes: number
}

function lectureBody(input: LectureInput): Record<string, unknown> {
  return {
    lecturer_id: input.lecturerId,
    event_id: input.eventId,
    name: input.name,
    duration: minutesToDuration(input.durationMinutes),
  }
}

/* Command side — lecture-service:50054. Creating a lecture kicks off the saga
   and the RabbitMQ mail worker that notifies the lecturer. */

export async function createLecture(
  input: LectureInput,
): Promise<LectureCommandResult> {
  const raw = await rpc('/api/lectures', lectureBody(input))
  return mapLecture(asObject(raw).lecture)
}

export async function updateLecture(
  id: number,
  input: LectureInput,
): Promise<void> {
  await rpc('/api/lectures/update', { id, ...lectureBody(input) })
}

export async function deleteLecture(
  id: number,
): Promise<LectureCommandResult> {
  const raw = await rpc('/api/lectures/delete', { id })
  return mapLecture(asObject(raw).lecture)
}

/* Query side — lecture-query-service:50055 */

export async function getLectureById(id: number): Promise<LectureRecord> {
  const raw = await rpc('/api/lectures/get-by-id', { id })
  return mapLectureQuery(asObject(raw).lecture)
}

export async function getLectureByName(name: string): Promise<LectureRecord> {
  const raw = await rpc('/api/lectures/get-by-name', { name })
  return mapLectureQuery(asObject(raw).lecture)
}

export async function listLecturesByEvent(
  eventId: number,
  page: number,
  pageSize: number,
  signal?: AbortSignal,
): Promise<Page<LectureRecord>> {
  const raw = await rpc(
    '/api/lectures/list-by-event',
    { event_id: eventId, page, page_size: pageSize },
    signal,
  )
  return mapPage(raw, 'lectures', mapLectureQuery)
}

export async function listLecturesByLecturer(
  lecturerId: number,
  page: number,
  pageSize: number,
  signal?: AbortSignal,
): Promise<Page<LectureRecord>> {
  const raw = await rpc(
    '/api/lectures/list-by-lecturer',
    { lecturer_id: lecturerId, page, page_size: pageSize },
    signal,
  )
  return mapPage(raw, 'lectures', mapLectureQuery)
}
