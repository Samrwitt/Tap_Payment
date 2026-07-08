# Tap Payment (Ethiopia-ready) — Monorepo

This repo contains:

- `backend/`: Go payments API (Tap-style flow: create charge → redirect → webhook → status/refund)
- `openapi/`: OpenAPI contract (source of truth for SDK generation)
- `sdk-typescript/`: TypeScript SDK generated from OpenAPI (added after API stabilizes)

## Quick start (backend)

### Prereqs

- Go 1.22+

### Configure env

Copy and edit:

```bash
cp backend/.env.example backend/.env
```

### Run

```bash
cd backend
go run ./cmd/api
```

Server runs on `http://localhost:8080` by default.

## Endpoints (draft)

- `POST /api/payments/charges`
- `POST /api/payments/webhooks/tap`
- `GET /api/payments/{paymentId}`

## Notes

Tap Payments does not currently list Ethiopia as an onboarding country for merchants. This project follows Tap’s API patterns and keeps an internal provider abstraction so Ethiopian gateways can be added later without changing the app-facing API.

