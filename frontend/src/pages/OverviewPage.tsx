import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { eventsApi, lecturersApi, locationsApi } from '../api'
import { Card, ErrorState, PageHeader, Spinner } from '../components/ui'

/**
 * Counts come from the `total_count` on each list envelope, so a single
 * one-row page is enough — no need to pull the collections themselves.
 */
function useTotals() {
  const lecturers = useQuery({
    queryKey: ['overview', 'lecturers'],
    queryFn: ({ signal }) =>
      lecturersApi.listLecturers({ page: 1, pageSize: 1 }, signal),
  })
  const locations = useQuery({
    queryKey: ['overview', 'locations'],
    queryFn: ({ signal }) =>
      locationsApi.listLocations({ page: 1, pageSize: 1 }, signal),
  })
  const events = useQuery({
    queryKey: ['overview', 'events'],
    queryFn: ({ signal }) =>
      eventsApi.listEvents({ page: 1, pageSize: 1 }, signal),
  })
  return { lecturers, locations, events }
}

const services = [
  {
    name: 'lecturer-service',
    port: '50051',
    role: 'Lecturers — commands and queries on one instance',
  },
  {
    name: 'event-service',
    port: '50052',
    role: 'Event commands, event sourcing, and locations',
  },
  {
    name: 'event-query-service',
    port: '50053',
    role: 'Event read model, projected over RabbitMQ',
  },
  {
    name: 'lecture-service',
    port: '50054',
    role: 'Lecture commands, saga orchestration, mail worker',
  },
  {
    name: 'lecture-query-service',
    port: '50055',
    role: 'Lecture read model, projected over RabbitMQ',
  },
]

export function OverviewPage() {
  const { lecturers, locations, events } = useTotals()
  const firstError = lecturers.error ?? locations.error ?? events.error

  return (
    <>
      <PageHeader
        title="Overview"
        description="An admin client for the MAIS microservice backend. Every request is a POST through the APISIX gateway, which transcodes JSON into gRPC before it reaches a service."
      />

      {firstError && (
        <div className="mb-6">
          <ErrorState
            error={firstError}
            onRetry={() => {
              void lecturers.refetch()
              void locations.refetch()
              void events.refetch()
            }}
          />
        </div>
      )}

      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <StatCard
          label="Lecturers"
          value={lecturers.data?.totalCount}
          loading={lecturers.isPending}
          to="/lecturers"
        />
        <StatCard
          label="Locations"
          value={locations.data?.totalCount}
          loading={locations.isPending}
          to="/locations"
        />
        <StatCard
          label="Events"
          value={events.data?.totalCount}
          loading={events.isPending}
          to="/events"
        />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card title="Services behind the gateway">
          <ul className="divide-y divide-slate-100">
            {services.map((service) => (
              <li key={service.name} className="py-2.5 first:pt-0 last:pb-0">
                <div className="flex items-baseline justify-between gap-3">
                  <span className="font-mono text-sm text-slate-900">
                    {service.name}
                  </span>
                  <span className="font-mono text-xs text-slate-400">
                    :{service.port}
                  </span>
                </div>
                <p className="mt-0.5 text-xs text-slate-500">{service.role}</p>
              </li>
            ))}
          </ul>
        </Card>

        <Card title="What to expect from this client">
          <ul className="space-y-3 text-sm text-slate-600">
            <li>
              <strong className="text-slate-900">Reads lag writes.</strong>{' '}
              Events and lectures are written on the command side and projected
              onto the query side asynchronously over RabbitMQ, so a list
              refreshed immediately after a write may still show the old row.
            </li>
            <li>
              <strong className="text-slate-900">
                Filters apply on submit.
              </strong>{' '}
              The gateway caps each client at 100 requests per minute, so the
              filter forms wait for an explicit Apply rather than firing on every
              keystroke.
            </li>
            <li>
              <strong className="text-slate-900">
                Timestamps travel as {'{seconds, nanos}'}.
              </strong>{' '}
              APISIX&apos;s transcoder does not implement proto3&apos;s canonical
              JSON mapping for the well-known types, so all conversion is
              centralised in <code className="font-mono text-xs">src/lib/codec.ts</code>.
            </li>
            <li>
              <strong className="text-slate-900">Every route is a POST.</strong>{' '}
              These protos carry no{' '}
              <code className="font-mono text-xs">google.api.http</code>{' '}
              annotations, so parameters travel in the body — even for lookups
              and deletes.
            </li>
          </ul>
        </Card>
      </div>
    </>
  )
}

function StatCard({
  label,
  value,
  loading,
  to,
}: {
  label: string
  value: number | undefined
  loading: boolean
  to: string
}) {
  return (
    <Link
      to={to}
      className="block rounded-lg bg-white p-5 shadow-sm ring-1 ring-slate-200 transition-shadow hover:shadow-md"
    >
      <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
        {label}
      </p>
      <p className="mt-2 text-3xl font-semibold text-slate-900">
        {loading ? (
          <Spinner className="size-6 text-slate-300" />
        ) : (
          (value ?? '—')
        )}
      </p>
    </Link>
  )
}
