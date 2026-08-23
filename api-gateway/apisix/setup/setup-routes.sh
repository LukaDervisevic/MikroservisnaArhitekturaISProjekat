#!/usr/bin/env bash
set -euo pipefail

ADMIN_URL="${APISIX_ADMIN_URL:-http://apisix:9180}"
ADMIN_KEY="${APISIX_ADMIN_KEY:?APISIX_ADMIN_KEY must be set}"
ROUTES_FILE="/etc/apisix-setup/routes.json"
PROTO_ROOT="/workspace"
TMP_DIR="$(mktemp -d)"

admin() {
  local method="$1" path="$2" data="${3:-}"
  local args=(-sS -o /tmp/resp.json -w "%{http_code}" -X "$method" "${ADMIN_URL}${path}" -H "X-API-KEY: ${ADMIN_KEY}" -H "Content-Type: application/json")
  if [[ -n "$data" ]]; then
    args+=(-d "$data")
  fi
  local code
  code="$(curl "${args[@]}")"
  if [[ "$code" -ge 300 ]]; then
    echo "FAILED ${method} ${path} -> HTTP ${code}" >&2
    cat /tmp/resp.json >&2
    exit 1
  fi
  echo "OK ${method} ${path} -> HTTP ${code}"
}

echo "==> Ensuring log directory is writable by the apisix container user..."
mkdir -p /workspace/logs
chmod 777 /workspace/logs

echo "==> Waiting for APISIX admin API..."
for i in $(seq 1 30); do
  if curl -sS -o /dev/null -w "%{http_code}" "${ADMIN_URL}/apisix/admin/routes" -H "X-API-KEY: ${ADMIN_KEY}" | grep -q "^200$"; then
    echo "Admin API is ready."
    break
  fi
  if [[ "$i" -eq 30 ]]; then
    echo "Admin API never became ready" >&2
    exit 1
  fi
  sleep 2
done

echo "==> Compiling proto FileDescriptorSets..."
cd "$PROTO_ROOT"
protoc -I . --include_imports --descriptor_set_out="${TMP_DIR}/lecturer.pb" proto/lecturer/lecturer.proto
protoc -I . --include_imports --descriptor_set_out="${TMP_DIR}/event.pb"    proto/event/event.proto
protoc -I . --include_imports --descriptor_set_out="${TMP_DIR}/lecture.pb" proto/lecture/lecture.proto

lecturer_b64="$(base64 -w0 "${TMP_DIR}/lecturer.pb")"
event_b64="$(base64 -w0 "${TMP_DIR}/event.pb")"
lecture_b64="$(base64 -w0 "${TMP_DIR}/lecture.pb")"

echo "==> Registering protos..."
admin PUT /apisix/admin/protos/lecturer-proto "$(jq -n --arg c "$lecturer_b64" '{content: $c}')"
admin PUT /apisix/admin/protos/event-proto    "$(jq -n --arg c "$event_b64" '{content: $c}')"
admin PUT /apisix/admin/protos/lecture-proto  "$(jq -n --arg c "$lecture_b64" '{content: $c}')"

echo "==> Generating self-signed TLS cert (CN=localhost)..."
openssl req -x509 -nodes -newkey rsa:2048 -days 825 \
  -keyout "${TMP_DIR}/tls.key" -out "${TMP_DIR}/tls.crt" \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1" 2>/dev/null

cert_content="$(cat "${TMP_DIR}/tls.crt")"
key_content="$(cat "${TMP_DIR}/tls.key")"

echo "==> Registering SSL certificate..."
admin PUT /apisix/admin/ssls/1 "$(jq -n --arg cert "$cert_content" --arg key "$key_content" \
  '{cert: $cert, key: $key, sni: "localhost"}')"

echo "==> Registering global rules (rate limiting + logging)..."
admin PUT /apisix/admin/global_rules/1 '{
  "plugins": {
    "limit-count": {
      "count": 100,
      "time_window": 60,
      "key_type": "var",
      "key": "remote_addr",
      "rejected_code": 429,
      "policy": "local"
    }
  }
}'

admin PUT /apisix/admin/global_rules/2 '{
  "plugins": {
    "file-logger": {
      "path": "/usr/local/apisix/logs/access/access.log"
    }
  }
}'

echo "==> Registering routes..."
route_count="$(jq 'length' "$ROUTES_FILE")"
for i in $(seq 0 $((route_count - 1))); do
  row="$(jq ".[$i]" "$ROUTES_FILE")"
  id="$(echo "$row" | jq -r '.id')"
  uri="$(echo "$row" | jq -r '.uri')"
  svc="$(echo "$row" | jq -r '.service')"
  method="$(echo "$row" | jq -r '.method')"
  proto="$(echo "$row" | jq -r '.proto')"
  upstream="$(echo "$row" | jq -r '.upstream')"

  plugins="$(jq -n --arg proto "$proto" --arg svc "$svc" --arg method "$method" \
    '{"grpc-transcode": {proto_id: $proto, service: $svc, method: $method}}')"

  route_body="$(jq -n --arg uri "$uri" --arg host "${upstream%%:*}" --argjson port "${upstream##*:}" --argjson plugins "$plugins" \
    '{uri: $uri, methods: ["POST"], plugins: $plugins, upstream: {scheme: "grpc", type: "roundrobin", nodes: {($host + ":" + ($port|tostring)): 1}}}')"

  admin PUT "/apisix/admin/routes/${id}" "$route_body"
done

echo "==> APISIX gateway setup complete: $route_count routes registered."
