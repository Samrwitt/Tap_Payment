# Tap Payments SDK (TypeScript)

Manual SDK matching `openapi/payments.yaml`.

## Usage

```ts
import { createApiClient } from "./src";

const api = createApiClient({
  baseUrl: "http://localhost:8080",
  adminApiKey: "dev-admin-key",
});

const payment = await api.createCharge({
  orderId: "ord_001",
  amount: 100,
  currency: "ETB",
  customer: {
    firstName: "Abebe",
    phone: { countryCode: "251", number: "911234567" },
  },
});

await api.getPayment(payment.paymentId);
await api.refundPayment(payment.paymentId, { reason: "customer request" });
```
