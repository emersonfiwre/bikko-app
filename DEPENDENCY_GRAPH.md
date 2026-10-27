# Backend dependency graph — 2026-10-26

## Composition and lifecycle

`cmd/bikkofy-api` and `cmd/bikkerfy-api` → `internal/application.Run` → validated `config.Config` → required PostgreSQL pool → `migrations.Apply` → optional explicit local fixture seed → repositories/services/controllers → `router.SetupRouter` → HTTP server with timeouts and graceful shutdown.

Both entrypoints expose the same modular API. The default local Compose runs one process on8080. No second gateway boundary is implied. Redis can be unavailable; PostgreSQL cannot fall back to sample data. No controller owns a data store or pushes synchronously.

## Route lookup

| Routes | Transport | Application/domain boundary | Persistence |
|---|---|---|---|
| GET /health, /ready | HealthController | liveness / database ping | required PostgreSQL |
| GET /home,/categories,/categories/:id,/services,/services/:id | HomeController | HomeUseCase | CategoryRepository, ServiceRepository, BikkerRepository; Redis category cache |
| GET /search | SearchController | SearchUseCase | categories/services, bounded queries |
| GET /bikkers,/bikkers/:id | BikkerController | public active profiles | BikkerRepository, services/reviews |
| POST /auth/check-phone,/auth/register,/auth/login | AuthController | AuthService + FirebaseTokenVerifier + NormalizePhone + JWTService | UserRepository; users/bikkers transaction |
| GET/PUT/DELETE /profile; PUT /profile/photo | ProfileController | ProfileService | UserRepository + ProfileRepository(NewProfileRepository) → model.SolicitationRepository |
| POST/DELETE /users/device-token | UserController | X-App-Type scope | optional UserDeviceRepository; user_devices |
| GET/POST /solicitations, GET /solicitations/:id, PATCH or POST /solicitations/:id/status | SolicitationController | model.SolicitationRepository + authenticated actor | SolicitationPostgresRepository, requests/negotiations/orders/outbox |
| GET /provider/feed,/provider/solicitations/:id; POST proposal/counter-offer/accept/reject | ProviderController | participant-scoped commands + professional role | same model.SolicitationRepository; row locks and idempotency |
| GET/POST/PUT/DELETE /provider/portfolio | ProviderController | ServiceMutationRepository, ownership | ServiceRepository transaction; services,photos,soft deletion |
| PUT /provider/profile | ProviderController | professional role separate from availability | BikkerProfileUpdater, users/bikkers/services transaction |
| POST /provider/evaluations | ProviderController | completed contract + participant | SolicitationPostgresRepository, client_reviews/ratings/outbox transaction |
| POST /reviews,/orders/:id/review,/solicitations/:id/review | ReviewController | RatingUseCase, actual JWT reviewer | ReviewRepository, completed order, ratings/outbox transaction |
| POST /storage/upload-url | StorageController | StorageUseCase, actor namespace/MIME/size | StorageService: explicit local disk or real R2 |
| PUT/GET /local-storage/*key | StorageController.LocalObject | signed capability, development only | persistent local Docker volume; actual bytes |
| GET /bootstrap | BootstrapController | feature map; payment disabled in MVP | optional Redis flags |

## Shared invariants

- AuthMiddleware validates JWT and reloads active account; `isBikker` is role, `isProviderAvailable` is availability. RequireProvider gates provider routes.
- Actor IDs and sender roles are never accepted as authority from JSON. Reads/commands restrict to customer or assigned provider.
- `model.SolicitationStatusUpdateInput` is command metadata, not a transport identity DTO. Domain errors live in `domain/contract_errors.go`.
- `orders.accepted_negotiation_id` fixes the accepted proposal. Money is normalized to exact decimal strings before SQL; client float values do not replace accepted prices.
- Quote row locks serialize transitions. Request keys + payload fingerprints handle creation retries; negotiation keys have scoped unique indexes. Five priced counter-offers; information messages do not count.
- Reputation rows, averages, evaluation flags and notification events are in the same transaction. Soft deletion preserves referenced work history.

## Notifications and storage

Contract/review transaction → notification_outbox → push.RunOutbox / deliverOne → FirebasePushService → UserDeviceRepository.GetDeviceTokens → FCM. Worker has SKIP LOCKED, retry/backoff and notification_id. No Firebase in local mode: keep events pending rather than pretending delivery.

StorageUseCase → typed image MIME/size + user namespace → StorageService.GenerateUploadURL. LOCAL_STORAGE=true + ENV=development → signed local PUT/GET and actual files; otherwise required R2 credentials and explicit errors. No mock URLs or upload success shortcut.

## Schema and maintenance commands

`migrations/*.up.sql` embedded in `migrations/migrate.go`: advisory lock, name/checksum ledger, legacy baseline, transaction before serving. `000007` replaces the duplicate historical password removal name. `000008` adds normalized active-phone uniqueness, contract idempotency/accepted proposal, user location, service deletion, devices, outbox and indexes.

`migrations.SeedLocalAccounts`: two named local accounts only; never inserts orders/portfolio and never resets completed test work. `cmd/reset-local-accounts --confirm-reset`: explicit development-only return of those accounts to empty data. `cmd/migrate`: same runner for administration.

## Tests

Unit: config, phone, auth/profile, JWT, middleware, controllers and repositories. PostGIS: contract_integration_test.go and identity_integration_test.go (transactions, isolation, concurrency, ownership, five rounds, canonical money, devices/deletion). Migrations integration covers fresh schema, legacy upgrade, repeated apply and exactly two empty local fixtures. Outbox integration isolates schema and verifies success, retry redaction, concurrency, cancellation and rollback. CI provisions PostGIS and sets TEST_DATABASE_URL; absent env is an explicit skip locally.
