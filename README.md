# Bikkofy backend — local development

A modular Go monolith serves the Bikkofy customer app and Bikkerfy professional app. PostgreSQL/PostGIS is the source of truth. Redis is optional cache; database failures are errors, never demonstration data. Both API entrypoints use the same composition in `internal/application`.

## Run locally (no R2 or Firebase credentials required for the two test accounts)

```sh
docker compose up -d --build app db redis
```

The local Compose explicitly enables `ENV=development`, `LOCAL_TEST_ACCOUNTS=true` and `LOCAL_STORAGE=true`. The API listens on localhost:8080; uploads are persisted in the Docker volume `bikko_local_media`. Each upload is a real HTTP PUT, checked for image type and limited to 10 MiB. No fake successful upload is returned.

Android uses `http://10.0.2.2:8080` on the emulator or `http://127.0.0.1:8080` on a device with `adb reverse tcp:8080 tcp:8080`. Localhost is intentionally retained in this testing phase, including the release variant. Turn off the Android “Ambiente Mock” datasource switch to test the real API. OTP bypass is restricted to debug builds, loopback and these exact accounts:

| App | Phone | Name | CPF | Email | Location |
|---|---|---|---|---|---|
| Bikkofy | 11999990000 | Lucas Ferreira Mendes | 345.678.901-22 | lucas.mendes@bikkofy.com.br | Pinheiros, São Paulo - SP (-23.5641, -46.6823) |
| Bikkerfy | 11988880000 | Rafael Oliveira Costa | 456.789.012-33 | rafael.costa@bikkerfy.com.br | Pinheiros, São Paulo - SP (-23.5614, -46.6914) |

New local fixtures contain zero requests, orders and portfolio items. Startup seeds only account/profile identity and preserves work from subsequent test sessions. Login exchanges `local_test_<phone>` for a real signed API JWT. Mock bearer tokens and other arbitrary test phones are rejected. Real Firebase identities always require a verified phone claim; registration/upgrade cannot select a user by a client-supplied phone.

To explicitly return only the two reserved accounts to the empty test baseline:

```sh
docker exec bikko-backend-app ./reset-local-accounts --confirm-reset
```

This deletes those accounts’ test requests, orders, evaluations, portfolio items, notification events and devices; preserves the users; and does not delete unrelated user accounts. It requires development mode and the explicit test-account setting. It never runs automatically at startup. Back up valuable local data before a reset.

## Schema and runtime

Migrations are embedded in the binary and applied under an advisory lock before serving HTTP. The ledger stores the complete filename and checksum and rejects edits after application. A legacy database initialized with the original SQL is baselined before incremental migrations. The duplicate historical password-removal migration is now `000007_remove_password_hash`; original installations had no repository migration ledger, so the runner adopts `000007` from schema state when the former migration already removed the column. Do not run a numeric-prefix external migrator over the legacy files without reconciling its own ledger first.

Use `docker exec bikko-backend-app ./migrate` to apply the same runner manually. Migration 000008 refuses duplicate active phone identities instead of merging or choosing an arbitrary account. There is no DDL or destructive data cleanup in repository constructors.

- `GET /api/v1/health`: process liveness.
- `GET /api/v1/ready`: PostgreSQL readiness; returns503 when unavailable.
- `POST /api/v1/auth/login`, `/register`: verified identity or the two explicit local fixtures.
- `/api/v1/provider/*`: authenticated professional role. Availability is separate: offline professionals can manage their profile and existing contracts.
- `POST /api/v1/solicitations` and negotiation mutations support `Idempotency-Key`; reusing a key with different data returns409.
- `PATCH /api/v1/solicitations/:id/status`: participant-scoped commands. Accept uses the last persisted proposal from the other party, never an arbitrary request price. Fifth price counter-offer remains acceptable; information requests do not consume a price round.
- Evaluations require explicit completion and correct participation; ratings and state changes are atomic.
- `POST /api/v1/storage/upload-url`: authenticated, scoped key, supported MIME and `size_bytes` required. Local signed PUT/GET URLs are issued by the development storage adapter.
- Device tokens are scoped by user and `X-App-Type: client|provider`, rather than one shared user token.

Notifications are inserted in `notification_outbox` in the business transaction. A worker with row locks/retry/backoff performs at-least-once delivery when Firebase messaging is configured; `notification_id` supports deduplication. With no Firebase in local mode, events stay persisted for inspection, and no delivery is falsely reported.

## Tests

```sh
go build ./...
go test ./...
TEST_DATABASE_URL='postgres://USER:PASSWORD@localhost:PORT/DISPOSABLE_DB?sslmode=disable' go test -race ./...
```

Without `TEST_DATABASE_URL`, PostgreSQL tests explicitly skip; do not interpret this as integration coverage. The workflow provisions PostgreSQL/PostGIS and runs all unit/integration tests with the race detector. Integration fixtures use independent identifiers/schemas; never point them at valuable data.

## Future cloud configuration

Cloud storage is not required for the present local setup. When enabled, set `LOCAL_STORAGE=false` and provide real R2 credentials/bucket configuration; missing credentials or signing errors fail explicitly. Local test identities/storage cannot be enabled outside development, and JWT secrets are required rather than silently defaulted. Private evidence uses time-limited signed read URLs; a durable media-key/refresh flow and verified private bucket policy must be completed before long-lived cloud media is relied on. Local signed read URLs expire after seven days and can be regenerated by uploading again during local testing.

HTTPS, real Firebase SMS/FCM, real R2 and release-on-device validation remain publication checks; localhost is an explicitly accepted local-testing setting.
