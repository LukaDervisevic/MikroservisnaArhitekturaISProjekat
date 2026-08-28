import { rpc, asObject } from './http'
import { mapLocation, mapPage, type Location, type Page } from './types'

export interface LocationInput {
  name: string
  address: string
  capacity: number
}

export interface ListLocationsParams {
  page: number
  pageSize: number
  /** Optional capacity filters; 0 means "no bound". */
  minCapacity?: number
  maxCapacity?: number
}

export async function createLocation(input: LocationInput): Promise<Location> {
  const raw = await rpc('/api/locations', {
    name: input.name,
    address: input.address,
    capacity: input.capacity,
  })
  return mapLocation(asObject(raw).location)
}

export async function updateLocation(
  id: number,
  input: LocationInput,
): Promise<void> {
  await rpc('/api/locations/update', {
    id,
    name: input.name,
    address: input.address,
    capacity: input.capacity,
  })
}

export async function deleteLocation(id: number): Promise<Location> {
  const raw = await rpc('/api/locations/delete', { id })
  return mapLocation(asObject(raw).location)
}

export async function getLocationById(id: number): Promise<Location> {
  const raw = await rpc('/api/locations/get-by-id', { id })
  return mapLocation(asObject(raw).location)
}

export async function getLocationByName(name: string): Promise<Location> {
  const raw = await rpc('/api/locations/get-by-name', { name })
  return mapLocation(asObject(raw).location)
}

export async function listLocations(
  params: ListLocationsParams,
  signal?: AbortSignal,
): Promise<Page<Location>> {
  const raw = await rpc(
    '/api/locations/list',
    {
      page: params.page,
      page_size: params.pageSize,
      min_capacity: params.minCapacity ?? 0,
      max_capacity: params.maxCapacity ?? 0,
    },
    signal,
  )
  return mapPage(raw, 'locations', mapLocation)
}

export async function listLocationsByMinCapacity(
  minCapacity: number,
  page: number,
  pageSize: number,
  signal?: AbortSignal,
): Promise<Page<Location>> {
  const raw = await rpc(
    '/api/locations/list-by-min-capacity',
    { min_capacity: minCapacity, page, page_size: pageSize },
    signal,
  )
  return mapPage(raw, 'locations', mapLocation)
}
