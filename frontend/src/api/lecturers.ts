import { rpc, asObject } from './http'
import { mapLecturer, mapPage, type Lecturer, type Page } from './types'

export interface LecturerInput {
  fullName: string
  title: string
  fieldOfExpertise: string
  email: string
}

export interface ListLecturersParams {
  page: number
  pageSize: number
  /** Server-side filters; omit or leave empty to disable. */
  fieldOfExpertise?: string
  title?: string
}

export async function createLecturer(input: LecturerInput): Promise<Lecturer> {
  const raw = await rpc('/api/lecturers', {
    full_name: input.fullName,
    title: input.title,
    field_of_expertise: input.fieldOfExpertise,
    email: input.email,
  })
  return mapLecturer(asObject(raw).lecturer)
}

export async function updateLecturer(
  id: number,
  input: LecturerInput,
): Promise<void> {
  await rpc('/api/lecturers/update', {
    id,
    full_name: input.fullName,
    title: input.title,
    field_of_expertise: input.fieldOfExpertise,
    email: input.email,
  })
}

export async function deleteLecturer(id: number): Promise<Lecturer> {
  const raw = await rpc('/api/lecturers/delete', { id })
  return mapLecturer(asObject(raw).lecturer)
}

export async function getLecturerById(id: number): Promise<Lecturer> {
  const raw = await rpc('/api/lecturers/get-by-id', { id })
  return mapLecturer(asObject(raw).lecturer)
}

export async function getLecturerByName(fullName: string): Promise<Lecturer> {
  const raw = await rpc('/api/lecturers/get-by-name', { full_name: fullName })
  return mapLecturer(asObject(raw).lecturer)
}

export async function listLecturers(
  params: ListLecturersParams,
  signal?: AbortSignal,
): Promise<Page<Lecturer>> {
  const raw = await rpc(
    '/api/lecturers/list',
    {
      page: params.page,
      page_size: params.pageSize,
      field_of_expertise: params.fieldOfExpertise ?? '',
      title: params.title ?? '',
    },
    signal,
  )
  return mapPage(raw, 'lecturers', mapLecturer)
}

export async function listLecturersByField(
  fieldOfExpertise: string,
  page: number,
  pageSize: number,
  signal?: AbortSignal,
): Promise<Page<Lecturer>> {
  const raw = await rpc(
    '/api/lecturers/list-by-field',
    { field_of_expertise: fieldOfExpertise, page, page_size: pageSize },
    signal,
  )
  return mapPage(raw, 'lecturers', mapLecturer)
}
