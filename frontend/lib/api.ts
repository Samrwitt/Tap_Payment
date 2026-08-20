const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

export type Phone = { countryCode: string; number: string };

export type Customer = {
  firstName?: string;
  lastName?: string;
  email?: string;
  phone: Phone;
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

export type OneTapResult = {
  paymentId: string;
  provider: string;
  providerChargeId: string;
  redirectUrl: string;
  status: string;
};

export type PaymentStatus = {
  id: string;
  orderId: string;
  provider: string;
  providerPaymentId: string;
  status: string;
  redirectUrl?: string;
  createdAt: string;
  updatedAt: string;
};

type ApiErrorBody = {
  error?: { code?: string; message?: string };
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {}),
    },
  });

  if (!res.ok) {
    let message = `Request failed (${res.status})`;
    try {
      const body = (await res.json()) as ApiErrorBody;
      if (body.error?.message) message = body.error.message;
    } catch {
      // ignore
    }
    throw new Error(message);
  }

  return (await res.json()) as T;
}

export function savePaymentMethod(body: {
  customerKey?: string;
  methodType: "wallet" | "card";
  cardNumber?: string;
  customer: Customer;
}) {
  return request<PaymentMethod>("/api/payments/methods", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function listPaymentMethods(customerKey: string) {
  return request<{ methods: PaymentMethod[] }>(
    `/api/payments/methods?customerKey=${encodeURIComponent(customerKey)}`,
  );
}

export function oneTapPay(body: {
  orderId: string;
  amount: number;
  currency: string;
  paymentMethodId: string;
}) {
  return request<OneTapResult>("/api/payments/one-tap", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function getPayment(paymentId: string) {
  return request<PaymentStatus>(`/api/payments/${encodeURIComponent(paymentId)}`);
}

export function customerKeyFromPhone(countryCode: string, number: string) {
  return `+${countryCode.replace(/^\+/, "")}${number}`;
}
