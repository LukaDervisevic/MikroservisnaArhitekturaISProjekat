import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { lecturesApi, type LectureRecord } from '../api'
import { useEventOptions, useLecturerOptions } from '../hooks/useOptions'
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
  Th,
} from '../components/ui'
import { durationToMinutes } from '../lib/codec'
import { formatDateTime, formatDuration, formatPrice } from '../lib/format'

const PAGE_SIZE = 10

/**
 * There is no "list all lectures" route — lecture-query-service only exposes
 * ListLecturesByEventID and ListLecturesByLecturerID — so the page always
 * scopes to one parent.
 */
type Scope = 'event' | 'lecturer'

const emptyForm: lecturesApi.LectureInput = {
  lecturerId: 0,
  eventId: 0,
  name: '',
  durationMinutes: 60,
}

export function LecturesPage() {
  const queryClient = useQueryClient()
  const events = useEventOptions()
  const lecturers = useLecturerOptions()

  const [scope, setScope] = useState<Scope>('event')
  const [parentId, setParentId] = useState(0)
  const [page, setPage] = useState(1)

  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<LectureRecord | null>(null)
  const [deleting, setDeleting] = useState<LectureRecord | null>(null)

  const routePath =
    scope === 'event'
      ? '/api/lectures/list-by-event'
      : '/api/lectures/list-by-lecturer'

  const listQuery = useQuery({
    queryKey: ['lectures', scope, parentId, page],
    // Both routes reject a zero parent id with InvalidArgument, so don't ask.
    enabled: parentId > 0,
    queryFn: ({ signal }) =>
      scope === 'event'
        ? lecturesApi.listLecturesByEvent(parentId, page, PAGE_SIZE, signal)
        : lecturesApi.listLecturesByLecturer(parentId, page, PAGE_SIZE, signal),
  })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['lectures'] })
  }

  const deleteMutation = useMutation({
    mutationFn: (record: LectureRecord) => lecturesApi.deleteLecture(record.id),
    onSuccess: () => {
      setDeleting(null)
      invalidate()
    },
  })

  const parentOptions =
    scope === 'event'
      ? (events.data ?? []).map((item) => ({
          id: item.id,
          label: `${item.name} · ${formatDateTime(item.dateTime)}`,
        }))
      : (lecturers.data ?? []).map((item) => ({
          id: item.id,
          label: item.title
            ? `${item.title} ${item.fullName}`
            : item.fullName,
        }))

  return (
    <>
      <PageHeader
        title="Lectures"
        description="Creating a lecture starts the saga in lecture-service:50054, which validates the lecturer and event before committing and queues the lecturer's notification e-mail over RabbitMQ."
        actions={<Button onClick={() => setCreating(true)}>New lecture</Button>}
      />

      <Card
        title="Scope"
        actions={<RouteBadge path={routePath} />}
        className="mb-6"
      >
        <div className="mb-4 flex gap-2">
          <Button
            variant={scope === 'event' ? 'primary' : 'secondary'}
            onClick={() => {
              setScope('event')
              setParentId(0)
              setPage(1)
            }}
          >
            By event
          </Button>
          <Button
            variant={scope === 'lecturer' ? 'primary' : 'secondary'}
            onClick={() => {
              setScope('lecturer')
              setParentId(0)
              setPage(1)
            }}
          >
            By lecturer
          </Button>
        </div>

        <Field
          label={scope === 'event' ? 'Event' : 'Lecturer'}
          hint="The query service has no unscoped list route, so a parent must be chosen."
        >
          <Select
            value={parentId || ''}
            onChange={(event) => {
              setParentId(Number(event.target.value))
              setPage(1)
            }}
          >
            <option value="">
              Select {scope === 'event' ? 'an event' : 'a lecturer'}…
            </option>
            {parentOptions.map((option) => (
              <option key={option.id} value={option.id}>
                {option.label}
              </option>
            ))}
          </Select>
        </Field>
      </Card>

      <Card title="Results">
        {parentId === 0 ? (
          <EmptyState
            message={`Choose ${scope === 'event' ? 'an event' : 'a lecturer'} above to load its lectures.`}
          />
        ) : listQuery.isPending ? (
          <LoadingState />
        ) : listQuery.isError ? (
          <ErrorState
            error={listQuery.error}
            onRetry={() => void listQuery.refetch()}
          />
        ) : listQuery.data.items.length === 0 ? (
          <EmptyState message="No lectures found for this selection." />
        ) : (
          <>
            <Table>
              <thead>
                <tr>
                  <Th className="w-16">ID</Th>
                  <Th>Lecture</Th>
                  <Th>Duration</Th>
                  <Th>Lecturer</Th>
                  <Th>Event</Th>
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
                      {record.name || '—'}
                    </Td>
                    <Td className="whitespace-nowrap">
                      {formatDuration(record.durationSeconds)}
                    </Td>
                    <Td>
                      <span className="block text-slate-900">
                        {record.lecturerTitle
                          ? `${record.lecturerTitle} ${record.lecturerFullName}`
                          : record.lecturerFullName || '—'}
                      </span>
                      {record.lecturerFieldOfExpertise && (
                        <span className="block text-xs text-slate-500">
                          {record.lecturerFieldOfExpertise}
                        </span>
                      )}
                    </Td>
                    <Td>
                      <span className="block text-slate-900">
                        {record.eventName || '—'}
                      </span>
                      {record.eventType && (
                        <Badge tone="indigo">{record.eventType}</Badge>
                      )}
                    </Td>
                    <Td className="whitespace-nowrap">
                      {formatDateTime(record.eventDateTime)}
                    </Td>
                    <Td>
                      <span className="block text-slate-900">
                        {record.locationName || '—'}
                      </span>
                      <span className="block text-xs text-slate-500">
                        {record.locationAddress}
                      </span>
                    </Td>
                    <Td>{formatPrice(record.cotisationPrice)}</Td>
                    <Td className="text-right">
                      <div className="flex justify-end gap-1">
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
        <Modal open title="New lecture" onClose={() => setCreating(false)}>
          <LectureForm
            initial={{
              ...emptyForm,
              ...(scope === 'event'
                ? { eventId: parentId }
                : { lecturerId: parentId }),
            }}
            onCancel={() => setCreating(false)}
            onSubmit={async (values) => {
              await lecturesApi.createLecture(values)
              setCreating(false)
              invalidate()
            }}
          />
        </Modal>
      )}

      {editing && (
        <Modal
          open
          title={`Edit lecture #${editing.id}`}
          onClose={() => setEditing(null)}
        >
          <LectureForm
            initial={{
              lecturerId: editing.lecturerId,
              eventId: editing.eventId,
              name: editing.name,
              durationMinutes: durationToMinutes({
                seconds: editing.durationSeconds,
              }),
            }}
            onCancel={() => setEditing(null)}
            onSubmit={async (values) => {
              await lecturesApi.updateLecture(editing.id, values)
              setEditing(null)
              invalidate()
            }}
          />
        </Modal>
      )}

      <ConfirmDialog
        open={deleting !== null}
        title="Delete lecture"
        message={`Delete “${deleting?.name ?? ''}” by ${deleting?.lecturerFullName ?? 'this lecturer'}?`}
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

function LectureForm({
  initial,
  onCancel,
  onSubmit,
}: {
  initial: lecturesApi.LectureInput
  onCancel: () => void
  onSubmit: (values: lecturesApi.LectureInput) => Promise<void>
}) {
  const [values, setValues] = useState(initial)
  const events = useEventOptions()
  const lecturers = useLecturerOptions()
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
      <Field label="Lecturer">
        <Select
          required
          value={values.lecturerId || ''}
          onChange={(event) =>
            setValues({ ...values, lecturerId: Number(event.target.value) })
          }
        >
          <option value="">Select a lecturer…</option>
          {(lecturers.data ?? []).map((lecturer) => (
            <option key={lecturer.id} value={lecturer.id}>
              {lecturer.title
                ? `${lecturer.title} ${lecturer.fullName}`
                : lecturer.fullName}
            </option>
          ))}
        </Select>
      </Field>
      <Field label="Event">
        <Select
          required
          value={values.eventId || ''}
          onChange={(event) =>
            setValues({ ...values, eventId: Number(event.target.value) })
          }
        >
          <option value="">Select an event…</option>
          {(events.data ?? []).map((item) => (
            <option key={item.id} value={item.id}>
              {item.name} · {formatDateTime(item.dateTime)}
            </option>
          ))}
        </Select>
      </Field>
      <Field
        label="Duration (minutes)"
        hint="Sent as a {seconds, nanos} protobuf Duration."
      >
        <Input
          type="number"
          min={1}
          required
          value={values.durationMinutes}
          onChange={(event) =>
            setValues({
              ...values,
              durationMinutes: Number(event.target.value),
            })
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
