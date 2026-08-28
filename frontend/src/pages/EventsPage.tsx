import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { eventsApi, type EventRecord } from '../api'
import { useLocationOptions } from '../hooks/useOptions'
import { ConfirmDialog } from '../components/ConfirmDialog'
import { RouteBadge } from '../components/RouteBadge'
import {
  Badge,
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
  Select,
  Table,
  Td,
  Textarea,
  Th,
} from '../components/ui'
import { dateToDatetimeLocal } from '../lib/codec'
import { formatDateTime, formatPrice } from '../lib/format'

const PAGE_SIZE = 10

type ListMode = 'list' | 'by-type'

interface Filters {
  type: string
  fromDate: string
  toDate: string
  locationId: string
}

const emptyFilters: Filters = {
  type: '',
  fromDate: '',
  toDate: '',
  locationId: '',
}

const emptyForm: eventsApi.EventInput = {
  name: '',
  cotisationPrice: 0,
  agenda: '',
  type: '',
  dateTime: null,
  locationId: 0,
}

export function EventsPage() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const locations = useLocationOptions()

  const [mode, setMode] = useState<ListMode>('list')
  const [page, setPage] = useState(1)
  const [draft, setDraft] = useState<Filters>(emptyFilters)
  const [filters, setFilters] = useState<Filters>(emptyFilters)

  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<EventRecord | null>(null)
  const [deleting, setDeleting] = useState<EventRecord | null>(null)

  const routePath =
    mode === 'list' ? '/api/events/list' : '/api/events/list-by-type'

  const listQuery = useQuery({
    queryKey: ['events', mode, page, filters],
    queryFn: ({ signal }) =>
      mode === 'list'
        ? eventsApi.listEvents(
            {
              page,
              pageSize: PAGE_SIZE,
              type: filters.type,
              fromDate: filters.fromDate ? new Date(filters.fromDate) : null,
              toDate: filters.toDate ? new Date(filters.toDate) : null,
              locationId: Number(filters.locationId) || 0,
            },
            signal,
          )
        : eventsApi.listEventsByType(filters.type, page, PAGE_SIZE, signal),
  })

  // Commands hit event-service, which runs an orchestrated saga that updates the
  // event-query-service projection before the write returns — so a refetch right
  // after a successful write already sees the change (a failed saga rolls it all
  // back). A tiny window still exists between the saga's DB commits.
  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['events'] })
  }

  const deleteMutation = useMutation({
    mutationFn: (record: EventRecord) => eventsApi.deleteEvent(record.id),
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
        title="Events"
        description="Writes go to event-service:50052 as one event-sourced aggregate: each runs an orchestrated saga that propagates to the event-query-service:50053 read model and lecture-service before returning. Use History to inspect an event's stream."
        actions={<Button onClick={() => setCreating(true)}>New event</Button>}
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
            List (all filters)
          </Button>
          <Button
            variant={mode === 'by-type' ? 'primary' : 'secondary'}
            onClick={() => {
              setMode('by-type')
              setPage(1)
            }}
          >
            By type
          </Button>
        </div>

        <form
          onSubmit={applyFilters}
          className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4"
        >
          <Field label="Type">
            <Input
              value={draft.type}
              onChange={(event) =>
                setDraft({ ...draft, type: event.target.value })
              }
              placeholder="e.g. Conference"
            />
          </Field>
          <Field
            label="From"
            hint={mode === 'by-type' ? 'Ignored by this route' : undefined}
          >
            <Input
              type="datetime-local"
              value={draft.fromDate}
              disabled={mode === 'by-type'}
              onChange={(event) =>
                setDraft({ ...draft, fromDate: event.target.value })
              }
            />
          </Field>
          <Field
            label="To"
            hint={mode === 'by-type' ? 'Ignored by this route' : undefined}
          >
            <Input
              type="datetime-local"
              value={draft.toDate}
              disabled={mode === 'by-type'}
              onChange={(event) =>
                setDraft({ ...draft, toDate: event.target.value })
              }
            />
          </Field>
          <Field
            label="Location"
            hint={mode === 'by-type' ? 'Ignored by this route' : undefined}
          >
            <Select
              value={draft.locationId}
              disabled={mode === 'by-type'}
              onChange={(event) =>
                setDraft({ ...draft, locationId: event.target.value })
              }
            >
              <option value="">Any location</option>
              {(locations.data ?? []).map((location) => (
                <option key={location.id} value={location.id}>
                  {location.name}
                </option>
              ))}
            </Select>
          </Field>
          <div className="flex items-end gap-2">
            <Button type="submit">Apply</Button>
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                setDraft(emptyFilters)
                setFilters(emptyFilters)
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
          <EmptyState message="No events match these filters." />
        ) : (
          <>
            <Table>
              <thead>
                <tr>
                  <Th className="w-16">ID</Th>
                  <Th>Name</Th>
                  <Th>Type</Th>
                  <Th>Date &amp; time</Th>
                  <Th>Location</Th>
                  <Th>Fee</Th>
                  <Th className="text-right">Actions</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {listQuery.data.items.map((record) => (
                  <tr key={record.id}>
                    <Td className="font-mono text-xs">{record.id}</Td>
                    <Td className="font-medium text-slate-900">
                      {record.name}
                    </Td>
                    <Td>
                      {record.type ? (
                        <Badge tone="indigo">{record.type}</Badge>
                      ) : (
                        '—'
                      )}
                    </Td>
                    <Td className="whitespace-nowrap">
                      {formatDateTime(record.dateTime)}
                    </Td>
                    <Td>
                      <span className="block text-slate-900">
                        {record.locationName || '—'}
                      </span>
                      <span className="block text-xs text-slate-500">
                        {record.locationAddress}
                        {record.locationCapacity
                          ? ` · seats ${record.locationCapacity.toLocaleString()}`
                          : ''}
                      </span>
                    </Td>
                    <Td>{formatPrice(record.cotisationPrice)}</Td>
                    <Td className="text-right">
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost"
                          onClick={() =>
                            navigate(`/sourced-events?id=${record.id}`)
                          }
                        >
                          History
                        </Button>
                        <Button
                          variant="ghost"
                          onClick={() => setEditing(record)}
                        >
                          Edit
                        </Button>
                        <Button
                          variant="ghost"
                          className="text-rose-600 hover:bg-rose-50"
                          onClick={() => setDeleting(record)}
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
        <Modal open title="New event" onClose={() => setCreating(false)}>
          <EventForm
            initial={emptyForm}
            onCancel={() => setCreating(false)}
            onSubmit={async (values) => {
              await eventsApi.createEvent(values)
              setCreating(false)
              invalidate()
            }}
          />
        </Modal>
      )}

      {editing && (
        <Modal
          open
          title={`Edit event #${editing.id}`}
          onClose={() => setEditing(null)}
        >
          <EventForm
            initial={{
              name: editing.name,
              cotisationPrice: editing.cotisationPrice,
              agenda: editing.agenda,
              type: editing.type,
              dateTime: editing.dateTime,
              locationId: editing.locationId,
            }}
            onCancel={() => setEditing(null)}
            onSubmit={async (values) => {
              await eventsApi.updateEvent(editing.id, values)
              setEditing(null)
              invalidate()
            }}
          />
        </Modal>
      )}

      <ConfirmDialog
        open={deleting !== null}
        title="Delete event"
        message={`Delete “${deleting?.name ?? ''}”? This cancels the event: it's removed here and from every read model in one saga, but the cancellation stays in its append-only history. Fails if the event still has lectures.`}
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

function EventForm({
  initial,
  onCancel,
  onSubmit,
}: {
  initial: eventsApi.EventInput
  onCancel: () => void
  onSubmit: (values: eventsApi.EventInput) => Promise<void>
}) {
  const [values, setValues] = useState(initial)
  const locations = useLocationOptions()
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
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Type">
          <Input
            value={values.type}
            placeholder="e.g. Workshop"
            onChange={(event) =>
              setValues({ ...values, type: event.target.value })
            }
          />
        </Field>
        <Field label="Cotisation price">
          <Input
            type="number"
            min={0}
            step="0.01"
            value={values.cotisationPrice}
            onChange={(event) =>
              setValues({
                ...values,
                cotisationPrice: Number(event.target.value),
              })
            }
          />
        </Field>
      </div>
      <Field
        label="Date & time"
        hint="Sent as a {seconds, nanos} protobuf Timestamp — the transcoder rejects RFC3339 strings."
      >
        <Input
          type="datetime-local"
          value={dateToDatetimeLocal(values.dateTime)}
          onChange={(event) =>
            setValues({
              ...values,
              dateTime: event.target.value
                ? new Date(event.target.value)
                : null,
            })
          }
        />
      </Field>
      <Field label="Location">
        <Select
          required
          value={values.locationId || ''}
          onChange={(event) =>
            setValues({ ...values, locationId: Number(event.target.value) })
          }
        >
          <option value="">Select a location…</option>
          {(locations.data ?? []).map((location) => (
            <option key={location.id} value={location.id}>
              {location.name} · seats {location.capacity.toLocaleString()}
            </option>
          ))}
        </Select>
      </Field>
      <Field label="Agenda">
        <Textarea
          rows={4}
          value={values.agenda}
          onChange={(event) =>
            setValues({ ...values, agenda: event.target.value })
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
