# Tap Payments SDK (TypeScript)

This is a manual SDK matching `openapi/payments.yaml`.

## Usage

```ts
import { createApiClient } from "@tap-payment/sdk";

const api = createApiClient({ baseUrl: "http://localhost:8080" });

const payment = await api.createCharge({
  orderId: "ord_001",
  amount: 1.0,
  currency: "SAR",
  customer: {
    firstName: "Test",
    lastName: "User",
    email: "test@example.com",
    phone: { countryCode: "966", number: "51234567" },
  },
});

// payment.redirectUrl -> redirect the customer to complete payment
```

## Webhook (server-side)

```ts
await api.tapWebhook({
  hashstring: "<value from Tap header>",
  body: <Tap payload you received>,
});
```

