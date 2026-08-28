import { useState, type ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { sourcedEventsApi, type SourcedEventEntry } from '../api'
import { useKnownAggregates } from '../hooks/useKnownAggregates'
import { useLocationOptions } from '../hooks/useOptions'
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
  Select,
  Textarea,
} from '../components/ui'
import { dateToDatetimeLocal } from '../lib/codec'
import { formatDateTime, formatDetails, formatPrice } from '../lib/format'

/** The six mutating commands, plus the snapshot maintenance operation. */
type Command =
  | 'rename'
  | 'reschedule'
  | 'relocate'
  | 'change-price'
  | 'cancel'

const commandLabels: Record<Command, string> = {
  rename: 'Rename',
  reschedule: 'Reschedule',
  relocate: 'Relocate',
  'change-price': 'Change price',
  cancel: 'Cancel event',
}

export function SourcedEventsPage() {
  const queryClient = useQueryClient()
  const { aggregates, remember, forget } = useKnownAggregates()

  const [aggregateId, setAggregateId] = useState('')
  const [idDraft, setIdDraft] = useState('')
  const [creating, setCreating] = useState(false)
  const [command, setCommand] = useState<Command | null>(null)

  const stateQuery = useQuery({
    queryKey: ['sourced-event', aggregateId, 'state'],
    enabled: aggregateId !== '',
    queryFn: ({ signal }) =>
      sourcedEventsApi.getSourcedEventState(aggregateId, signal),
  })

  const historyQuery = useQuery({
    queryKey: ['sourced-event', aggregateId, 'history'],
    enabled: aggregateId !== '',
    queryFn: ({ signal }) =>
      sourcedEventsApi.getSourcedEventHistory(aggregateId, signal),
  })

  const refresh = () => {
    void queryClient.invalidateQueries({
      queryKey: ['sourced-event', aggregateId],
    })
  }

  const snapshotMutation = useMutation({
    mutationFn: () => sourcedEventsApi.createSourcedEventSnapshot(aggregateId),
    onSuccess: refresh,
  })

  const load = (id: string) => {
    const trimmed = id.trim()
    if (!trimmed) return
    setAggregateId(trimmed)
    setIdDraft(trimmed)
  }

  return (
    <>
      <PageHeader
        title="Event sourcing"
        description="The event-sourced write side of event-service. Each command appends an immutable event to the aggregate's stream and returns the new version; state is rebuilt by replaying that stream from the latest snapshot."
        actions={
          <Button onClick={() => setCreating(true)}>New sourced event</Button>
        }
      />

      <Card title="Aggregate" className="mb-6">
        <form
          className="flex flex-wrap items-end gap-3"
          onSubmit={(event) => {
            event.preventDefault()
            load(idDraft)
          }}
        >
          <div className="min-w-64 flex-1">
            <Field
              label="Aggregate ID"
              hint="There is no route to list aggregates, so IDs created here are remembered in this browser only."
            >
              <Input
                value={idDraft}
                onChange={(event) => setIdDraft(event.target.value)}
                placeholder="e.g. 8f14e45f-ceea-467a-9f36-8d1f7b2c0a13"
                className="font-mono"
              />
            </Field>
          </div>
          <Button type="submit">Load</Button>
        </form>

        {aggregates.length > 0 && (
          <div className="mt-4 border-t border-slate-200 pt-4">
            <p className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-500">
              Known aggregates
            </p>
            <ul className="flex flex-wrap gap-2">
              {aggregates.map((entry) => (
                <li key={entry.id}>
                  <span
                    className={`inline-flex items-center gap-2 rounded-md px-2 py-1 text-xs ring-1 ring-inset ${
                      entry.id === aggregateId
                        ? 'bg-indigo-50 text-indigo-700 ring-indigo-200'
                        : 'bg-white text-slate-600 ring-slate-300'
                    }`}
                  >
                    <button
                      type="button"
                      onClick={() => load(entry.id)}
                      className="max-w-56 truncate font-medium"
                      title={entry.id}
                    >
                      {entry.label || entry.id}
                    </button>
                    <button
                      type="button"
                      onClick={() => forget(entry.id)}
                      aria-label={`Forget ${entry.label || entry.id}`}
                      className="text-slate-400 hover:text-slate-700"
                    >
                      ×
                    </button>
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </Card>

      {aggregateId === '' ? (
        <Card>
          <EmptyState message="Load an aggregate ID, or create a new sourced event, to inspect its state and history." />
        </Card>
      ) : (
        <div className="grid gap-6 lg:grid-cols-5">
          <div className="lg:col-span-2">
            <Card
              title="Current state"
              actions={
                <Button
                  variant="secondary"
                  onClick={refresh}
                  loading={stateQuery.isFetching || historyQuery.isFetching}
                >
                  Refresh
                </Button>
              }
            >
              {stateQuery.isPending ? (
                <LoadingState />
              ) : stateQuery.isError ? (
                <ErrorState
                  error={stateQuery.error}
                  onRetry={() => void stateQuery.refetch()}
                />
              ) : (
                <>
                  <dl className="space-y-3">
                    <Detail label="Name">
                      <span className="font-medium text-slate-900">
                        {stateQuery.data.name || '—'}
                      </span>
                    </Detail>
                    <Detail label="Version">
                      <Badge tone="indigo">v{stateQuery.data.version}</Badge>
                    </Detail>
                    <Detail label="Status">
                      {stateQuery.data.cancelled ? (
                        <Badge tone="rose">Cancelled</Badge>
                      ) : (
                        <Badge tone="emerald">Active</Badge>
                      )}
                    </Detail>
                    <Detail label="Type">{stateQuery.data.type || '—'}</Detail>
                    <Detail label="Date & time">
                      {formatDateTime(stateQuery.data.dateTime)}
                    </Detail>
                    <Detail label="Cotisation price">
                      {formatPrice(stateQuery.data.cotisationPrice)}
                    </Detail>
                    <Detail label="Location ID">
                      <span className="font-mono text-xs">
                        {stateQuery.data.locationId || '—'}
                      </span>
                    </Detail>
                    <Detail label="Agenda">
                      <span className="whitespace-pre-wrap text-sm">
                        {stateQuery.data.agenda || '—'}
                      </span>
                    </Detail>
                  </dl>

                  <div className="mt-5 border-t border-slate-200 pt-4">
                    <p className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-500">
                      Commands
                    </p>
                    <div className="flex flex-wrap gap-2">
                      {(Object.keys(commandLabels) as Command[]).map((key) => (
                        <Button
                          key={key}
                          variant={key === 'cancel' ? 'danger' : 'secondary'}
                          disabled={stateQuery.data.cancelled}
                          onClick={() => setCommand(key)}
                        >
                          {commandLabels[key]}
                        </Button>
                      ))}
                    </div>
                    {stateQuery.data.cancelled && (
                      <p className="mt-2 text-xs text-slate-500">
                        A cancelled aggregate rejects further commands.
                      </p>
                    )}

                    <div className="mt-4 border-t border-slate-200 pt-4">
                      <Button
                        variant="secondary"
                        loading={snapshotMutation.isPending}
                        onClick={() => snapshotMutation.mutate()}
                      >
                        Create snapshot
                      </Button>
                      <p className="mt-2 text-xs text-slate-500">
                        Collapses the replay baseline to the current version.
                        Does not append an event, so the history below is
                        unchanged.
                      </p>
                      {snapshotMutation.isError && (
                        <div className="mt-3">
                          <ErrorState error={snapshotMutation.error} />
                        </div>
                      )}
                      {snapshotMutation.isSuccess && (
                        <p className="mt-2 text-xs text-emerald-700">
                          Snapshot taken at v{snapshotMutation.data.version}.
                        </p>
                      )}
                    </div>
                  </div>
                </>
              )}
            </Card>
          </div>

          <div className="lg:col-span-3">
            <Card title="History">
              {historyQuery.isPending ? (
                <LoadingState />
              ) : historyQuery.isError ? (
                <ErrorState
                  error={historyQuery.error}
                  onRetry={() => void historyQuery.refetch()}
                />
              ) : historyQuery.data.length === 0 ? (
                <EmptyState message="This aggregate has no recorded events." />
              ) : (
                <Timeline entries={historyQuery.data} />
              )}
            </Card>
          </div>
        </div>
      )}

      {creating && (
        <Modal open title="New sourced event" onClose={() => setCreating(false)}>
          <CreateSourcedEventForm
            onCancel={() => setCreating(false)}
            onCreated={(id, label) => {
              remember(id, label)
              setCreating(false)
              load(id)
            }}
          />
        </Modal>
      )}

      {command && (
        <CommandModal
          command={command}
          aggregateId={aggregateId}
          onClose={() => setCommand(null)}
          onDone={() => {
            setCommand(null)
            refresh()
          }}
        />
      )}
    </>
  )
}

function Detail({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">
        {label}
      </dt>
      <dd className="mt-0.5 text-slate-700">{children}</dd>
    </div>
  )
}

function Timeline({ entries }: { entries: SourcedEventEntry[] }) {
  // Newest first reads better for an audit log, and the array arrives in
  // append order.
  const ordered = [...entries].sort((a, b) => b.version - a.version)
  return (
    <ol className="relative space-y-5 border-l border-slate-200 pl-5">
      {ordered.map((entry) => (
        <li key={entry.eventId || `${entry.aggregateId}-${entry.version}`}>
          <span className="absolute -left-[5px] mt-1.5 size-2.5 rounded-full bg-indigo-500 ring-4 ring-white" />
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-medium text-slate-900">
              {entry.eventType || 'Unknown event'}
            </span>
            <Badge tone="slate">v{entry.version}</Badge>
            <span className="text-xs text-slate-500">
              {formatDateTime(entry.occurredAt)}
            </span>
          </div>
          {entry.details && (
            <pre className="mt-2 overflow-x-auto rounded-md bg-slate-50 p-3 font-mono text-xs text-slate-700 ring-1 ring-slate-200">
              {formatDetails(entry.details)}
            </pre>
          )}
        </li>
      ))}
    </ol>
  )
}

function CreateSourcedEventForm({
  onCancel,
  onCreated,
}: {
  onCancel: () => void
  onCreated: (aggregateId: string, label: string) => void
}) {
  const locations = useLocationOptions()
  const [values, setValues] = useState({
    name: '',
    cotisationPrice: 0,
    agenda: '',
    type: '',
    dateTime: null as Date | null,
    locationId: 0,
  })

  const mutation = useMutation({
    mutationFn: () => sourcedEventsApi.createSourcedEvent(values),
    onSuccess: (result) => onCreated(result.aggregateId, values.name),
  })

  return (
    <form
      className="space-y-4"
      onSubmit={(event) => {
        event.preventDefault()
        mutation.mutate()
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
      <Field label="Date & time">
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
              {location.name}
            </option>
          ))}
        </Select>
      </Field>
      <Field label="Agenda">
        <Textarea
          rows={3}
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
          Create
        </Button>
      </div>
    </form>
  )
}

function CommandModal({
  command,
  aggregateId,
  onClose,
  onDone,
}: {
  command: Command
  aggregateId: string
  onClose: () => void
  onDone: () => void
}) {
  const locations = useLocationOptions()
  const [text, setText] = useState('')
  const [number, setNumber] = useState(0)
  const [dateTime, setDateTime] = useState('')

  const mutation = useMutation({
    mutationFn: () => {
      switch (command) {
        case 'rename':
          return sourcedEventsApi.renameSourcedEvent(aggregateId, text)
        case 'reschedule':
          return sourcedEventsApi.rescheduleSourcedEvent(
            aggregateId,
            new Date(dateTime),
          )
        case 'relocate':
          return sourcedEventsApi.relocateSourcedEvent(aggregateId, number)
        case 'change-price':
          return sourcedEventsApi.changeSourcedEventPrice(aggregateId, number)
        case 'cancel':
          return sourcedEventsApi.cancelSourcedEvent(aggregateId, text)
      }
    },
    onSuccess: onDone,
  })

  return (
    <Modal open title={commandLabels[command]} onClose={onClose}>
      <form
        className="space-y-4"
        onSubmit={(event) => {
          event.preventDefault()
          mutation.mutate()
        }}
      >
        {command === 'rename' && (
          <Field label="New name">
            <Input
              required
              value={text}
              onChange={(event) => setText(event.target.value)}
            />
          </Field>
        )}

        {command === 'cancel' && (
          <Field
            label="Reason"
            hint="Recorded on the cancellation event; the aggregate rejects further commands afterwards."
          >
            <Textarea
              rows={3}
              required
              value={text}
              onChange={(event) => setText(event.target.value)}
            />
          </Field>
        )}

        {command === 'reschedule' && (
          <Field label="New date & time">
            <Input
              type="datetime-local"
              required
              value={dateTime}
              onChange={(event) => setDateTime(event.target.value)}
            />
          </Field>
        )}

        {command === 'relocate' && (
          <Field label="New location">
            <Select
              required
              value={number || ''}
              onChange={(event) => setNumber(Number(event.target.value))}
            >
              <option value="">Select a location…</option>
              {(locations.data ?? []).map((location) => (
                <option key={location.id} value={location.id}>
                  {location.name} · seats{' '}
                  {location.capacity.toLocaleString()}
                </option>
              ))}
            </Select>
          </Field>
        )}

        {command === 'change-price' && (
          <Field label="New cotisation price">
            <Input
              type="number"
              min={0}
              step="0.01"
              required
              value={number}
              onChange={(event) => setNumber(Number(event.target.value))}
            />
          </Field>
        )}

        {mutation.isError && <ErrorState error={mutation.error} />}

        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant={command === 'cancel' ? 'danger' : 'primary'}
            loading={mutation.isPending}
          >
            {commandLabels[command]}
          </Button>
        </div>
      </form>
    </Modal>
  )
}
