# MAIS Admin — frontend

A React + TypeScript admin client for the MAIS microservice backend. It talks
only to the APISIX gateway, which transcodes JSON into gRPC before requests
reach any service — the frontend never speaks gRPC and never needs generated
stubs.

## Running it

The backend must be up first:

```bash
docker compose up -d          # from the repository root
```

Then:

```bash
cd frontend
npm install
npm run dev                   # http://localhost:5173
```

`npm run build` produces a static bundle in `dist/`, `npm run lint` runs
ESLint, and `npm run typecheck` runs `tsc` without emitting.

## Why there is a dev proxy

APISIX has no `cors` plugin in its global rules, so a browser on `:5173`
calling `:9080` directly is blocked at preflight. Rather than change the
gateway, `vite.config.ts` proxies same-origin `/api/*` to
`http://localhost:9080`. Override the target with `VITE_GATEWAY_URL`.

This is a **development** arrangement. Serving `dist/` from a static host
would need either a reverse proxy in front of both, or a `cors` global rule
added to `api-gateway/apisix/setup/setup-routes.sh`.

## Talking to the gateway

Three properties of the transcoder shape the whole API layer. All of them are
documented in `api-gateway/README.md`; the client contains them in one place so
they never reach a component.

1. **Every route is a POST with parameters in the body.** These protos carry no
   `google.api.http` annotations, so `grpc-transcode` cannot lift path or query
   parameters into fields — even ID lookups and deletes send a JSON body. There
   is therefore exactly one request shape, in `src/api/http.ts`.

2. **`Content-Type: application/json` is mandatory.** Without it the transcoder
   skips JSON parsing and every field silently arrives empty, surfacing as a
   confusing `400`.

3. **`Timestamp` and `Duration` travel as `{seconds, nanos}`.** APISIX 3.14.1
   does not implement proto3's canonical JSON mapping for the well-known types;
   an RFC3339 string crashes it with a `500`. Every conversion lives in
   `src/lib/codec.ts`.

Responses come back with the proto field names as declared — snake_case, except
`GetEventByIdQueryResponse.eventWithLocation`, which is literally declared in
camelCase in `event.proto`. `src/api/types.ts` maps each wire message onto a
camelCase domain type with real `Date`s, so the quirks stop at that boundary.

## Layout

```
src/
  api/          one function per gateway route, grouped by service
    http.ts     the single POST helper + ApiError
    types.ts    domain types and wire -> domain mappers
  lib/
    codec.ts    Timestamp/Duration <-> Date/minutes
    format.ts   display formatting
  components/   UI primitives, layout, dialogs
  hooks/        dropdown sources, remembered aggregate IDs
  pages/        one page per domain
```

## Behaviour worth knowing

- **Reads lag writes.** Events and lectures are written on the command side and
  projected onto the query side asynchronously over RabbitMQ. A list refetched
  immediately after a write can still show the previous state. The UI
  invalidates its caches on mutation but cannot make the projection faster.
- **Filters apply on submit, not on keystroke.** The gateway's global
  `limit-count` rule allows 100 requests per minute per IP; debounced live
  filtering would burn that quota. A `429` is surfaced with a specific message
  and is never retried.
- **Several resources have two list routes** — a filterable `list` and a
  narrower dedicated one. Both are reachable, and the badge on each page shows
  which one produced the rows on screen.
- **Lectures must be scoped to a parent.** `lecture-query-service` exposes only
  `ListLecturesByEventID` and `ListLecturesByLecturerID`; there is no unscoped
  list route.
- **`LectureQuery` carries the lecture's own name and duration** via the
  `lecture_name` / `lecture_duration` fields (proto field numbers 16 and 17).
  Older builds of the backend omit them; the mapper degrades to an empty name
  and a zero duration rather than failing.
- **Aggregate IDs are remembered locally.** `event-service` has no route to
  enumerate event-sourced aggregates, so IDs created in this UI are kept in
  `localStorage` as a convenience. They are not a source of truth.

## Known backend gaps this client works around

- `CreateEvent` returns an `Event` whose nested `location` is not populated, so
  the create response alone cannot render venue details. The mapper degrades
  gracefully and the list refetch fills them in.
