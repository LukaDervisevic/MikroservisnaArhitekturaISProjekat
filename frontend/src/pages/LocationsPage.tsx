import { useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { locationsApi, type Location } from '../api'
import { ConfirmDialog } from '../components/ConfirmDialog'
import { RouteBadge } from '../components/RouteBadge'
import {
  Button,
  Card,
  EmptyState,
  ErrorState,
  Field,
  Input,
  LoadingState,
  Modal,
  PageHeader,
  Pagination,
  Table,
  Td,
  Th,
} from '../components/ui'

const PAGE_SIZE = 10

type ListMode = 'list' | 'by-min-capacity'

interface Filters {
  minCapacity: string
  maxCapacity: string
}

const emptyForm: locationsApi.LocationInput = {
  name: '',
  address: '',
  capacity: 0,
}

export function LocationsPage() {
  const queryClient = useQueryClient()

  const [mode, setMode] = useState<ListMode>('list')
  const [page, setPage] = useState(1)
  const [draft, setDraft] = useState<Filters>({
    minCapacity: '',
    maxCapacity: '',
  })
  const [filters, setFilters] = useState<Filters>(draft)

  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<Location | null>(null)
  const [deleting, setDeleting] = useState<Location | null>(null)

  // The proto uses 0 as "no bound" rather than an optional field.
  const minCapacity = Number(filters.minCapacity) || 0
  const maxCapacity = Number(filters.maxCapacity) || 0

  const routePath =
    mode === 'list'
      ? '/api/locations/list'
      : '/api/locations/list-by-min-capacity'

  const listQuery = useQuery({
    queryKey: ['locations', mode, page, minCapacity, maxCapacity],
    queryFn: ({ signal }) =>
      mode === 'list'
        ? locationsApi.listLocations(
            { page, pageSize: PAGE_SIZE, minCapacity, maxCapacity },
            signal,
          )
        : locationsApi.listLocationsByMinCapacity(
            minCapacity,
            page,
            PAGE_SIZE,
            signal,
          ),
  })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['locations'] })
  }

  const deleteMutation = useMutation({
    mutationFn: (location: Location) => locationsApi.deleteLocation(location.id),
    onSuccess: () => {
      setDeleting(null)
      invalidate()
    },
  })

  const applyFilters = (event: FormEvent) => {
    event.preventDefault()
    setPage(1)
    setFilters(draft)
  }

  return (
    <>
      <PageHeader
        title="Locations"
        description="Venues owned by event-service:50052. Events reference a location by ID, and the query-side read models denormalise the venue's name, address and capacity into every event row."
        actions={<Button onClick={() => setCreating(true)}>New location</Button>}
      />

      <Card
        title="Filters"
        actions={<RouteBadge path={routePath} />}
        className="mb-6"
      >
        <div className="mb-4 flex gap-2">
          <Button
            variant={mode === 'list' ? 'primary' : 'secondary'}
            onClick={() => {
              setMode('list')
              setPage(1)
            }}
          >
            List (min + max)
          </Button>
          <Button
            variant={mode === 'by-min-capacity' ? 'primary' : 'secondary'}
            onClick={() => {
              setMode('by-min-capacity')
              setPage(1)
            }}
          >
            By minimum capacity
          </Button>
        </div>

        <form
          onSubmit={applyFilters}
          className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3"
        >
          <Field label="Minimum capacity" hint="Leave empty for no bound">
            <Input
              type="number"
              min={0}
              value={draft.minCapacity}
              onChange={(event) =>
                setDraft({ ...draft, minCapacity: event.target.value })
              }
            />
          </Field>
          <Field
            label="Maximum capacity"
            hint={
              mode === 'by-min-capacity'
                ? 'Ignored by the by-min-capacity route'
                : 'Leave empty for no bound'
            }
          >
            <Input
              type="number"
              min={0}
              value={draft.maxCapacity}
              disabled={mode === 'by-min-capacity'}
              onChange={(event) =>
                setDraft({ ...draft, maxCapacity: event.target.value })
              }
            />
          </Field>
          <div className="flex items-end gap-2">
            <Button type="submit">Apply</Button>
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                const cleared = { minCapacity: '', maxCapacity: '' }
                setDraft(cleared)
                setFilters(cleared)
                setPage(1)
              }}
            >
              Reset
            </Button>
          </div>
        </form>
      </Card>

      <Card title="Results">
        {listQuery.isPending ? (
          <LoadingState />
        ) : listQuery.isError ? (
          <ErrorState
            error={listQuery.error}
            onRetry={() => void listQuery.refetch()}
          />
        ) : listQuery.data.items.length === 0 ? (
          <EmptyState message="No locations match these filters." />
        ) : (
          <>
            <Table>
              <thead>
                <tr>
                  <Th className="w-16">ID</Th>
                  <Th>Name</Th>
                  <Th>Address</Th>
                  <Th>Capacity</Th>
                  <Th className="text-right">Actions</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {listQuery.data.items.map((location) => (
                  <tr key={location.id}>
                    <Td className="font-mono text-xs">{location.id}</Td>
                    <Td className="font-medium text-slate-900">
                      {location.name}
                    </Td>
                    <Td>{location.address || '—'}</Td>
                    <Td>{location.capacity.toLocaleString()}</Td>
                    <Td className="text-right">
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost"
                          onClick={() => setEditing(location)}
                        >
                          Edit
                        </Button>
                        <Button
                          variant="ghost"
                          className="text-rose-600 hover:bg-rose-50"
                          onClick={() => setDeleting(location)}
                        >
                          Delete
                        </Button>
                      </div>
                    </Td>
                  </tr>
                ))}
              </tbody>
            </Table>
            <Pagination
              page={listQuery.data.page || page}
              pageSize={listQuery.data.pageSize || PAGE_SIZE}
              totalCount={listQuery.data.totalCount}
              hasNextPage={listQuery.data.hasNextPage}
              onPageChange={setPage}
            />
          </>
        )}
      </Card>

      {creating && (
        <Modal open title="New location" onClose={() => setCreating(false)}>
          <LocationForm
            initial={emptyForm}
            onCancel={() => setCreating(false)}
            onSubmit={async (values) => {
              await locationsApi.createLocation(values)
              setCreating(false)
              invalidate()
            }}
          />
        </Modal>
      )}

      {editing && (
        <Modal
          open
          title={`Edit location #${editing.id}`}
          onClose={() => setEditing(null)}
        >
          <LocationForm
            initial={{
              name: editing.name,
              address: editing.address,
              capacity: editing.capacity,
            }}
            onCancel={() => setEditing(null)}
            onSubmit={async (values) => {
              await locationsApi.updateLocation(editing.id, values)
              setEditing(null)
              invalidate()
            }}
          />
        </Modal>
      )}

      <ConfirmDialog
        open={deleting !== null}
        title="Delete location"
        message={`Delete “${deleting?.name ?? ''}”? Events still pointing at this location are not cascaded by the backend.`}
        pending={deleteMutation.isPending}
        error={deleteMutation.error}
        onCancel={() => {
          setDeleting(null)
          deleteMutation.reset()
        }}
        onConfirm={() => deleting && deleteMutation.mutate(deleting)}
      />
    </>
  )
}

function LocationForm({
  initial,
  onCancel,
  onSubmit,
}: {
  initial: locationsApi.LocationInput
  onCancel: () => void
  onSubmit: (values: locationsApi.LocationInput) => Promise<void>
}) {
  const [values, setValues] = useState(initial)
  const mutation = useMutation({ mutationFn: onSubmit })

  return (
    <form
      className="space-y-4"
      onSubmit={(event) => {
        event.preventDefault()
        mutation.mutate(values)
      }}
    >
      <Field label="Name">
        <Input
          required
          value={values.name}
          onChange={(event) =>
            setValues({ ...values, name: event.target.value })
          }
        />
      </Field>
      <Field label="Address">
        <Input
          value={values.address}
          onChange={(event) =>
            setValues({ ...values, address: event.target.value })
          }
        />
      </Field>
      <Field label="Capacity">
        <Input
          type="number"
          min={0}
          required
          value={values.capacity}
          onChange={(event) =>
            setValues({ ...values, capacity: Number(event.target.value) })
          }
        />
      </Field>

      {mutation.isError && <ErrorState error={mutation.error} />}

      <div className="flex justify-end gap-2 pt-2">
        <Button type="button" variant="secondary" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" loading={mutation.isPending}>
          Save
        </Button>
      </div>
    </form>
  )
}
