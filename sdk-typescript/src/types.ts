export type ErrorResponse = {
  error: {
    code: string;
    message: string;
  };
};

export type PhoneInput = {
  countryCode: string;
  number: string;
};

export type CustomerInput = {
  firstName?: string;
  lastName?: string;
  email?: string;
  phone: PhoneInput;
};

export type MetadataMap = Record<string, string>;

export type CreateChargeRequest = {
  orderId: string;
  amount: number;
  currency: string;
  customer: CustomerInput;
  metadata?: MetadataMap;
};

export type CreateChargeResponse = {
  paymentId: string;
  provider: string;
  providerChargeId: string;
  redirectUrl: string;
  status: string;
};

// Partial Tap charge webhook payload used by this backend for signature verification.
export type TapChargeWebhookPayload = {
  id: string;
  object: string;
  status: string;
  amount: number;
  currency: string;
  reference: {
    gateway: string;
    payment: string;
  };
  transaction: {
    created: string;
  };
};

export type WebhookOkResponse = { status: "ok" };
export type WebhookDuplicateResponse = { status: "duplicate" };

export type GetPaymentResponse = {
  id: string;
  orderId: string;
  provider: string;
  providerPaymentId: string;
  status: string;
  redirectUrl?: string | null;
  createdAt: string;
  updatedAt: string;
};

export type RefundRequest = {
  amount?: number;
  reason?: string;
};

export type RefundResponse = {
  paymentId: string;
  provider: string;
  providerRefundId: string;
  status: string;
};

export type SaveMethodRequest = {
  customerKey?: string;
  methodType: "wallet" | "card";
  cardNumber?: string;
  customer: CustomerInput;
};

export type PaymentMethod = {
  id: string;
  customerKey: string;
  provider: string;
  label: string;
  brand?: string;
  last4?: string;
  createdAt: string;
};

export type OneTapRequest = {
  orderId: string;
  amount: number;
  currency: string;
  paymentMethodId: string;
  metadata?: MetadataMap;
};

export type ChapaWebhookPayload = {
  tx_ref: string;
  status: string;
  reference?: string;
  amount?: unknown;
  currency?: string;
};

