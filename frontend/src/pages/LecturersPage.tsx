import { useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { lecturersApi, type Lecturer } from '../api'
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

/** Which of the two list routes to call. */
type ListMode = 'list' | 'by-field'

interface Filters {
  fieldOfExpertise: string
  title: string
}

const emptyForm: lecturersApi.LecturerInput = {
  fullName: '',
  title: '',
  fieldOfExpertise: '',
  email: '',
}

export function LecturersPage() {
  const queryClient = useQueryClient()

  const [mode, setMode] = useState<ListMode>('list')
  const [page, setPage] = useState(1)
  // Filters are only committed on submit rather than on every keystroke — the
  // gateway's global rule caps clients at 100 requests/minute.
  const [draft, setDraft] = useState<Filters>({
    fieldOfExpertise: '',
    title: '',
  })
  const [filters, setFilters] = useState<Filters>(draft)

  const [editing, setEditing] = useState<Lecturer | null>(null)
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState<Lecturer | null>(null)

  const routePath =
    mode === 'list' ? '/api/lecturers/list' : '/api/lecturers/list-by-field'

  const listQuery = useQuery({
    queryKey: ['lecturers', mode, page, filters],
    queryFn: ({ signal }) =>
      mode === 'list'
        ? lecturersApi.listLecturers(
            {
              page,
              pageSize: PAGE_SIZE,
              fieldOfExpertise: filters.fieldOfExpertise,
              title: filters.title,
            },
            signal,
          )
        : lecturersApi.listLecturersByField(
            filters.fieldOfExpertise,
            page,
            PAGE_SIZE,
            signal,
          ),
  })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['lecturers'] })
  }

  const deleteMutation = useMutation({
    mutationFn: (lecturer: Lecturer) =>
      lecturersApi.deleteLecturer(lecturer.id),
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

  const switchMode = (next: ListMode) => {
    setMode(next)
    setPage(1)
  }

  return (
    <>
      <PageHeader
        title="Lecturers"
        description="Backed by lecturer-service:50051, the one service that is not split into command and query halves — reads and writes both land on the same instance."
        actions={
          <Button onClick={() => setCreating(true)}>New lecturer</Button>
        }
      />

      <Card
        title="Filters"
        actions={<RouteBadge path={routePath} />}
        className="mb-6"
      >
        <div className="mb-4 flex gap-2">
          <Button
            variant={mode === 'list' ? 'primary' : 'secondary'}
            onClick={() => switchMode('list')}
          >
            List (field + title)
          </Button>
          <Button
            variant={mode === 'by-field' ? 'primary' : 'secondary'}
            onClick={() => switchMode('by-field')}
          >
            By field of expertise
          </Button>
        </div>

        <form
          onSubmit={applyFilters}
          className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3"
        >
          <Field label="Field of expertise">
            <Input
              value={draft.fieldOfExpertise}
              onChange={(event) =>
                setDraft({ ...draft, fieldOfExpertise: event.target.value })
              }
              placeholder="e.g. Distributed Systems"
            />
          </Field>
          <Field
            label="Title"
            hint={
              mode === 'by-field'
                ? 'Ignored by the by-field route'
                : undefined
            }
          >
            <Input
              value={draft.title}
              onChange={(event) =>
                setDraft({ ...draft, title: event.target.value })
              }
              placeholder="e.g. Professor"
              disabled={mode === 'by-field'}
            />
          </Field>
          <div className="flex items-end gap-2">
            <Button type="submit">Apply</Button>
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                const cleared = { fieldOfExpertise: '', title: '' }
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
          <EmptyState message="No lecturers match these filters." />
        ) : (
          <>
            <Table>
              <thead>
                <tr>
                  <Th className="w-16">ID</Th>
                  <Th>Full name</Th>
                  <Th>Title</Th>
                  <Th>Field of expertise</Th>
                  <Th>Email</Th>
                  <Th className="text-right">Actions</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {listQuery.data.items.map((lecturer) => (
                  <tr key={lecturer.id}>
                    <Td className="font-mono text-xs">{lecturer.id}</Td>
                    <Td className="font-medium text-slate-900">
                      {lecturer.fullName}
                    </Td>
                    <Td>{lecturer.title || '—'}</Td>
                    <Td>{lecturer.fieldOfExpertise || '—'}</Td>
                    <Td>{lecturer.email || '—'}</Td>
                    <Td className="text-right">
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost"
                          onClick={() => setEditing(lecturer)}
                        >
                          Edit
                        </Button>
                        <Button
                          variant="ghost"
                          className="text-rose-600 hover:bg-rose-50"
                          onClick={() => setDeleting(lecturer)}
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

      <LecturerFormModal
        open={creating}
        title="New lecturer"
        initial={emptyForm}
        onClose={() => setCreating(false)}
        onSubmit={async (values) => {
          await lecturersApi.createLecturer(values)
          setCreating(false)
          invalidate()
        }}
      />

      <LecturerFormModal
        open={editing !== null}
        title={`Edit lecturer #${editing?.id ?? ''}`}
        initial={
          editing
            ? {
                fullName: editing.fullName,
                title: editing.title,
                fieldOfExpertise: editing.fieldOfExpertise,
                email: editing.email,
              }
            : emptyForm
        }
        onClose={() => setEditing(null)}
        onSubmit={async (values) => {
          if (!editing) return
          await lecturersApi.updateLecturer(editing.id, values)
          setEditing(null)
          invalidate()
        }}
      />

      <ConfirmDialog
        open={deleting !== null}
        title="Delete lecturer"
        message={`Delete “${deleting?.fullName ?? ''}”? Lectures already referencing this lecturer are not cascaded by the backend.`}
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

function LecturerFormModal({
  open,
  title,
  initial,
  onClose,
  onSubmit,
}: {
  open: boolean
  title: string
  initial: lecturersApi.LecturerInput
  onClose: () => void
  onSubmit: (values: lecturersApi.LecturerInput) => Promise<void>
}) {
  // Remounting on open resets the form to `initial` without an effect.
  return open ? (
    <Modal open title={title} onClose={onClose}>
      <LecturerForm initial={initial} onCancel={onClose} onSubmit={onSubmit} />
    </Modal>
  ) : null
}

function LecturerForm({
  initial,
  onCancel,
  onSubmit,
}: {
  initial: lecturersApi.LecturerInput
  onCancel: () => void
  onSubmit: (values: lecturersApi.LecturerInput) => Promise<void>
}) {
  const [values, setValues] = useState(initial)
  const mutation = useMutation({ mutationFn: onSubmit })

  const update =
    (key: keyof lecturersApi.LecturerInput) =>
    (event: { target: { value: string } }) =>
      setValues((current) => ({ ...current, [key]: event.target.value }))

  return (
    <form
      className="space-y-4"
      onSubmit={(event) => {
        event.preventDefault()
        mutation.mutate(values)
      }}
    >
      <Field label="Full name">
        <Input required value={values.fullName} onChange={update('fullName')} />
      </Field>
      <Field label="Title">
        <Input value={values.title} onChange={update('title')} />
      </Field>
      <Field label="Field of expertise">
        <Input
          value={values.fieldOfExpertise}
          onChange={update('fieldOfExpertise')}
        />
      </Field>
      <Field
        label="Email"
        hint="Where lecture-service's RabbitMQ mail worker notifies this lecturer when a lecture is assigned."
      >
        <Input type="email" value={values.email} onChange={update('email')} />
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
