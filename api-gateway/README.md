# API Gateway (Apache APISIX)

This project fronts all 5 backend gRPC microservices with [Apache APISIX](https://apisix.apache.org/) 3.14.1, backed by etcd. APISIX terminates HTTP(S) from clients, transcodes JSON REST requests into gRPC calls against the backend services (`grpc-transcode` plugin), and applies rate limiting, TLS termination, and logging/monitoring — all via APISIX plugin configuration, with **no changes to backend service code**. (Response caching was attempted but is not functional for these routes — see [Response caching](#response-caching-not-supported) below for why.)

## Architecture

```
client (REST/JSON)
   |
   v
APISIX :9080 (HTTP) / :9443 (HTTPS, self-signed "localhost" cert)
   | grpc-transcode plugin: JSON body -> protobuf -> gRPC call
   v
lecturer-service:50051        (lecturer.LecturerService)
event-service:50052           (event.EventService [command], location.LocationService)
event-query-service:50053     (event.EventService [query])
lecture-service:50054         (lecture.LectureService [command])
lecture-query-service:50055   (lecture.LectureService [query])
```

APISIX config (`api-gateway/apisix/config.yaml`) enables the Admin API on `:9180` backed by etcd (`api-gateway/... -> etcd:2379`). All routes, protos, TLS cert, and plugin rules are pushed into etcd via the Admin API by a one-shot `apisix-setup` container (`api-gateway/apisix/setup/`) that runs automatically on `docker compose up`, once, after APISIX is healthy. Re-running it is safe (every call is an idempotent `PUT` with an explicit resource ID).

## Ports

| Port | Purpose |
|---|---|
| 9080 | HTTP traffic (REST endpoints below) |
| 9443 | HTTPS traffic (self-signed cert, SNI `localhost`) |
| 9180 | Admin API (route/plugin management) |
| 9091 | Prometheus metrics (`/apisix/prometheus/metrics`) |
| 9092 | Control API |

## Route mapping

Every route is `POST`, and the JSON request body maps 1:1 onto the corresponding proto request message's fields (field names as declared in the `.proto`, JSON camelCase or original proto field names both work via the transcoder's default options). This "RPC-style" mapping was chosen deliberately: `grpc-transcode` only translates a JSON **body** into a gRPC request — it does not lift URL path parameters into fields (these protos have no `google.api.http` annotations) — so every method, including ID/name lookups and deletes, takes its parameters in the body rather than the URL.

**Every request must send `Content-Type: application/json` explicitly.** `grpc-transcode` only parses the body as JSON when this header is present; without it (e.g. plain `curl -d '{"id":1}'`, which defaults to `application/x-www-form-urlencoded`) fields silently come through empty and you'll get a `400` with a `grpc-message` like `"full_name is required"`.

**`google.protobuf.Timestamp`/`Duration` fields must be sent as `{"seconds": N, "nanos": N}`, not as strings.** Proto3's canonical JSON mapping represents these as RFC3339 strings (`"2026-09-01T10:00:00Z"`) / duration strings (`"3600s"`), but APISIX 3.14.1's `grpc-transcode` doesn't implement that special-casing — it treats every message-typed field as a plain nested object, so a string value there crashes the transcoder (`500`, `bad argument #1 to 'isarray'`). Affected fields: `event.date_time` (`CreateEvent`/`UpdateEvent`) and `lecture.duration` (`CreateLecture`/`UpdateLecture`). Example: `"date_time": {"seconds": 1798106400, "nanos": 0}`.

| REST route | RPC | Upstream |
|---|---|---|
| `POST /api/lecturers` | `lecturer.LecturerService.CreateLecturer` | lecturer-service:50051 |
| `POST /api/lecturers/get-by-id` | `GetLecturerByID` `{"id": 1}` | lecturer-service:50051 |
| `POST /api/lecturers/get-by-name` | `GetLecturerByName` `{"full_name": "..."}` | lecturer-service:50051 |
| `POST /api/lecturers/list` | `ListLecturers` | lecturer-service:50051 |
| `POST /api/lecturers/list-by-field` | `ListLecturersByFieldOfExpertise` | lecturer-service:50051 |
| `POST /api/lecturers/update` | `UpdateLecturer` | lecturer-service:50051 |
| `POST /api/lecturers/delete` | `DeleteLecturer` `{"id": 1}` | lecturer-service:50051 |
| `POST /api/events` | `event.EventService.CreateEvent` | event-service:50052 |
| `POST /api/events/update` | `UpdateEvent` | event-service:50052 |
| `POST /api/events/delete` | `DeleteEvent` `{"id": 1}` | event-service:50052 |
| `POST /api/events/get-by-id` | `GetEventByID` `{"id": 1}` | event-query-service:50053 |
| `POST /api/events/get-by-name` | `GetEventByName` `{"name": "..."}` | event-query-service:50053 |
| `POST /api/locations` | `location.LocationService.CreateLocation` | event-service:50052 |
| `POST /api/locations/update` | `UpdateLocation` | event-service:50052 |
| `POST /api/locations/delete` | `DeleteLocation` `{"id": 1}` | event-service:50052 |
| `POST /api/locations/get-by-id` | `GetLocationByID` `{"id": 1}` | event-service:50052 |
| `POST /api/locations/get-by-name` | `GetLocationByName` `{"name": "..."}` | event-service:50052 |
| `POST /api/locations/list` | `ListLocations` | event-service:50052 |
| `POST /api/locations/list-by-min-capacity` | `ListLocationsByMinCapacity` | event-service:50052 |
| `POST /api/lectures` | `lecture.LectureService.CreateLecture` | lecture-service:50054 |
| `POST /api/lectures/update` | `UpdateLecture` | lecture-service:50054 |
| `POST /api/lectures/delete` | `DeleteLecture` `{"id": 1}` | lecture-service:50054 |
| `POST /api/lectures/get-by-id` | `GetLectureByID` `{"id": 1}` | lecture-query-service:50055 |
| `POST /api/lectures/get-by-name` | `GetLectureByName` `{"name": "..."}` | lecture-query-service:50055 |
| `POST /api/lectures/list-by-event` | `ListLecturesByEventID` `{"event_id": 1}` | lecture-query-service:50055 |
| `POST /api/lectures/list-by-lecturer` | `ListLecturesByLecturerID` `{"lecturer_id": 1}` | lecture-query-service:50055 |

The exact mapping lives in [`api-gateway/apisix/setup/routes.json`](./apisix/setup/routes.json) — that file is the source of truth; this table is generated from it.

**Not routed (not implemented server-side yet):**
- `event.EventService.ListEvents` / `ListEventsByType` — only `GetEventByID`/`GetEventByName` are implemented in `event-query-service` today.
- `lecturer.MailService.SendEmail` — dead code; mail is actually sent via a RabbitMQ worker in `lecture-service`, not gRPC.

Add these once their server-side implementations exist, by adding a row to `routes.json` and re-running the setup container (`docker compose up apisix-setup` or `docker compose run --rm apisix-setup`).

## gRPC-to-REST transcoding (`grpc-transcode`)

Because `event.proto` imports `location.proto`, and `lecture.proto` imports both `lecturer.proto` and `event.proto`, APISIX's plain-text proto registration (which can't resolve local file imports) isn't enough. Instead, the setup script compiles each proto into a self-contained `FileDescriptorSet` with all its dependencies inlined:

```bash
protoc -I . --include_imports --descriptor_set_out=event.pb proto/event/event.proto
```

...base64-encodes it, and registers it via `PUT /apisix/admin/protos/event-proto`. Three descriptor sets cover all 4 proto files:

- `lecturer-proto` — `lecturer.proto`
- `event-proto` — `event.proto` (also contains `location.LocationService`, since it's imported)
- `lecture-proto` — `lecture.proto` (also contains `event`, `location`, `lecturer` — only used for Lecture routes to keep the proto_id-to-service mapping easy to follow)

Each route's `grpc-transcode` plugin references one of these `proto_id`s plus the fully-qualified `service` and `method` name.

## Rate limiting & throttling

A global `limit-count` rule applies to every route uniformly (`api-gateway/apisix/setup/setup-routes.sh`, `global_rules/1`): **100 requests / 60s per client IP** (`remote_addr`), rejecting over-quota requests with `429 Too Many Requests`. Adjust `count`/`time_window` in the setup script and re-run it to change the limit.

## TLS termination

The setup container generates a self-signed certificate (`CN=localhost`, SAN `localhost`/`127.0.0.1`) on every run via `openssl` and registers it as an APISIX SSL object bound to SNI `localhost`. This is dev-only — connect with `curl -k` or otherwise trust/ignore the certificate warning. To use a real certificate instead, replace the `openssl req` step in `setup-routes.sh` with a `PUT /apisix/admin/ssls/1` using your real cert/key PEM content (or provision via a proper ACME/cert-manager flow — not needed for local dev).

## Response caching (not supported)

APISIX's `proxy-cache` plugin was attached to the read routes (`get-by-id`, `get-by-name`, `list*`) but turned out to be a **silent no-op** and was removed. Root cause, confirmed by reading APISIX 3.14.1's own nginx config template (`apisix/cli/ngx_tpl.lua`): the `proxy_cache`/`proxy_cache_key`/`proxy_no_cache`/`proxy_cache_bypass` directives only exist in the `location /` block that uses `proxy_pass` (plain HTTP upstreams). Routes with `upstream.scheme: grpc` — i.e. every route in this gateway — execute through a separate `location @grpc_pass` block using `grpc_pass`, which has no caching directives at all; `grpc-transcode` explicitly jumps execution there (`ngx.exec("@grpc_pass")`). The `proxy-cache` plugin still runs its access/header-filter phases and sets the relevant `$upstream_cache_*` variables, but nothing downstream ever reads them, so no `Apisix-Cache-Status` header is ever sent and nothing is ever written to a cache zone.

There's no plugin-level workaround — fixing this would mean patching APISIX's bundled nginx template (`ngx_tpl.lua`) to add caching support to the `@grpc_pass` location and building a custom image from that patch, which is outside the scope of gateway *configuration*. If response caching becomes a hard requirement, the options are: (a) carry that template patch, (b) build a small internal HTTP shim in front of grpc-transcode so `proxy_pass` (and its caching) is back in the path, or (c) cache at the client/BFF layer instead of the gateway.

## Logging & monitoring

- **Logging**: a global `file-logger` rule (`global_rules/2`) writes every request to `/usr/local/apisix/logs/access/access.log` inside the container, bind-mounted to `./api-gateway/apisix/logs/access.log` on the host.
- **Monitoring**: the `prometheus` plugin is enabled by default in APISIX; `api-gateway/apisix/config.yaml` overrides its export bind address to `0.0.0.0:9091` (the default `127.0.0.1` would be unreachable from the published Docker port). Scrape `http://localhost:9091/apisix/prometheus/metrics`. No Prometheus/Grafana containers are included in this repo — point your own Prometheus instance at that endpoint, or `curl` it directly for spot checks.

## Admin API security (dev-only, flagged not fixed)

`api-gateway/apisix/config.yaml` currently commits a plaintext Admin API key and sets `allow_admin: 0.0.0.0/0` (open to any IP), both explicitly marked "Dev only" in the file. This is fine for local development but **must not be used as-is in any shared/production environment** — rotate the key, restrict `allow_admin` to trusted CIDRs, and pull the key from a secret store rather than committing it.

## Regenerating routes / adding a new method

1. Add/update the RPC in the relevant `.proto` and implement it server-side.
2. Add a row to `api-gateway/apisix/setup/routes.json` (`id`, `uri`, `service`, `method`, `proto`, `upstream`).
3. Re-run the setup container: `docker compose up apisix-setup` (or `docker compose run --rm apisix-setup` if it already exited).

## Manual verification

```bash
# Create
curl -X POST localhost:9080/api/lecturers -H "Content-Type: application/json" -d '{"full_name":"Test","title":"Dr","field_of_expertise":"CS"}'

# Read
curl -X POST localhost:9080/api/lecturers/get-by-id -H "Content-Type: application/json" -d '{"id":1}'

# Over HTTPS (self-signed, -k to skip verification)
curl -k -X POST https://localhost:9443/api/lecturers/get-by-id -H "Content-Type: application/json" -d '{"id":1}'

# Rate limiting (watch X-RateLimit-Remaining count down, then 429 after 100/60s)
curl -i -X POST localhost:9080/api/lecturers/get-by-id -H "Content-Type: application/json" -d '{"id":1}' | grep -i ratelimit

# Metrics
curl localhost:9091/apisix/prometheus/metrics

# Logs
tail -f api-gateway/apisix/logs/access.log
```
