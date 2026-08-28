/**
 * Dropdown sources for the create/edit forms.
 *
 * None of the services expose a "fetch all" route, so these pull a single
 * generous page. Enough for a course project's data volume; a real deployment
 * would want a typeahead backed by the paginated routes instead.
 */

import { useQuery } from '@tanstack/react-query'
import { eventsApi, lecturersApi, locationsApi } from '../api'

const OPTIONS_PAGE_SIZE = 100

export function useLocationOptions() {
  return useQuery({
    queryKey: ['locations', 'options'],
    queryFn: ({ signal }) =>
      locationsApi.listLocations(
        { page: 1, pageSize: OPTIONS_PAGE_SIZE },
        signal,
      ),
    staleTime: 60_000,
    select: (page) => page.items,
  })
}

export function useLecturerOptions() {
  return useQuery({
    queryKey: ['lecturers', 'options'],
    queryFn: ({ signal }) =>
      lecturersApi.listLecturers(
        { page: 1, pageSize: OPTIONS_PAGE_SIZE },
        signal,
      ),
    staleTime: 60_000,
    select: (page) => page.items,
  })
}

export function useEventOptions() {
  return useQuery({
    queryKey: ['events', 'options'],
    queryFn: ({ signal }) =>
      eventsApi.listEvents({ page: 1, pageSize: OPTIONS_PAGE_SIZE }, signal),
    staleTime: 60_000,
    select: (page) => page.items,
  })
}
