# Bikko Backend Platform (Golang + PostgreSQL 16 + PostGIS + Redis)

High-concurrency backend API for **Bikko (Bikkofy)** gig economy handyman platform built with **Clean Architecture** in Golang.

## Architecture Highlights
- **Golang 1.22+**: Built with Gin web framework and `pgx/v5` connection pool.
- **Clean Architecture**: Decoupled layers (`domain`, `usecase`, `infrastructure/persistence`, `infrastructure/cache`, `infrastructure/http`).
- **PostgreSQL 16 + PostGIS 3.4**: Spatial indexing via `GiST` indexes on `location_geom` (`GEOGRAPHY(Point, 4326)`). Fast proximity calculations with `ST_DWithin` and spatial KNN (`<->`).
- **Redis 7 Cache Pipeline**: High-performance caching for dynamic Home feed (`home:categories`, `home:near:{geohash}`, `home:advice`).
- **Transactional Rating Cascade**: Atomic ACID transaction (`pgx.Tx`) updating `services.reviews_average` and recalculating `bikkers.rating` on completed orders.
- **Cloudflare R2 Storage**: S3-compatible pre-signed upload URL generation (`aws-sdk-go-v2`).

---

## Directory Structure
```
bikko-app/
├── cmd/api/
│   └── main.go                 # Application entrypoint & dependency injection
├── config/
│   └── config.go               # Environment variables & DSN configuration
├── internal/
│   ├── domain/                 # Domain entities & repository interfaces
│   ├── usecase/                # Business logic use cases (Home, Rating, Storage, Search)
│   └── infrastructure/
│       ├── persistence/postgres # pgxpool implementations & SQL queries
│       ├── cache/               # go-redis/v9 cache service & geohash pipeline
│       ├── storage/             # aws-sdk-go-v2 Cloudflare R2 S3 pre-signer
│       └── http/                # Gin controllers & router endpoints
├── migrations/
│   ├── 000001_init_schema.up.sql # PostGIS DDL schema migration
│   └── seed_data.sql            # Testing seed data
├── Dockerfile                   # Multi-stage build Dockerfile
└── docker-compose.yml           # PostgreSQL 16 + PostGIS, Redis 7 & Go app orchestration
```

---

## Quick Start & Local Setup

### 1. Run via Docker Compose
```bash
docker compose up -d
```

### 2. Seed Test Data into PostgreSQL PostGIS
```bash
docker exec -i bikko-postgres-db psql -U bikko -d bikkodb < migrations/seed_data.sql
```

### 3. API Endpoints
- `GET /` - Root health message
- `GET /api/v1/health` - Healthcheck
- `GET /api/v1/home?lat=-23.55052&lon=-46.63330` - Aggregated Home Feed (Redis Cache)
- `GET /api/v1/search?q=eletricista` - Search Services & Categories
- `POST /api/v1/reviews` - Post Order Rating (ACID Transaction)
- `POST /api/v1/storage/upload-url` - Generate Cloudflare R2 Upload Pre-signed URL

---

## Remote Git Repository
- Repository: [https://github.com/emersonfiwre/bikko-app.git](https://github.com/emersonfiwre/bikko-app.git)
