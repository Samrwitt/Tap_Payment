# BirrPay Frontend

Next.js checkout demo for the Tap Payment Ethiopia-ready backend.

## Run

Terminal 1 — backend:

```bash
cd backend
cp .env.example .env
go run ./cmd/api
```

Terminal 2 — frontend:

```bash
cd frontend
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).

Set API URL if needed:

```bash
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080 npm run dev
```

## Flow

1. Submit the payment form
2. Redirect to provider checkout (`mock` HTML page, or Tap/Chapa)
3. After pay, return to `/status/{paymentId}` (auto-polls status)
