import type {
  ChapaWebhookPayload,
  CreateChargeRequest,
  CreateChargeResponse,
  CustomerInput,
  ErrorResponse,
  GetPaymentResponse,
  MetadataMap,
  PhoneInput,
  RefundRequest,
  RefundResponse,
  TapChargeWebhookPayload,
  WebhookDuplicateResponse,
  WebhookOkResponse,
} from "./types";

export type ApiClientOptions = {
  baseUrl: string;
  adminApiKey?: string;
  fetchImpl?: typeof fetch;
};

export class ApiError extends Error {
  status: number;
  code: string;
  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

export function createApiClient(options: ApiClientOptions) {
  const fetchImpl = options.fetchImpl ?? fetch;

  async function request<T>(path: string, init: RequestInit): Promise<T> {
    const headers = new Headers(init.headers);
    if (options.adminApiKey && !headers.has("X-Admin-API-Key")) {
      headers.set("X-Admin-API-Key", options.adminApiKey);
    }
    const res = await fetchImpl(options.baseUrl.replace(/\/$/, "") + path, {
      ...init,
      headers,
    });
    const contentType = res.headers.get("content-type") ?? "";

    if (!res.ok) {
      let payload: ErrorResponse | null = null;
      if (contentType.includes("application/json")) {
        try {
          payload = (await res.json()) as ErrorResponse;
        } catch {
          // ignore parse errors
        }
      }
      const code = payload?.error?.code ?? "HTTP_ERROR";
      const message = payload?.error?.message ?? `Request failed with status ${res.status}`;
      throw new ApiError(res.status, code, message);
    }

    if (contentType.includes("application/json")) {
      return (await res.json()) as T;
    }
    return (await res.text()) as unknown as T;
  }

  return {
    async createCharge(body: CreateChargeRequest): Promise<CreateChargeResponse> {
      return request<CreateChargeResponse>("/api/payments/charges", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
    },

    async tapWebhook(params: {
      hashstring: string;
      body: TapChargeWebhookPayload;
    }): Promise<WebhookOkResponse | WebhookDuplicateResponse> {
      return request<WebhookOkResponse | WebhookDuplicateResponse>("/api/payments/webhooks/tap", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          hashstring: params.hashstring,
        },
        body: JSON.stringify(params.body),
      });
    },

    async chapaWebhook(params: {
      signature: string;
      body: ChapaWebhookPayload;
    }): Promise<WebhookOkResponse | WebhookDuplicateResponse> {
      return request<WebhookOkResponse | WebhookDuplicateResponse>("/api/payments/webhooks/chapa", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "x-chapa-signature": params.signature,
        },
        body: JSON.stringify(params.body),
      });
    },

    async getPayment(paymentId: string): Promise<GetPaymentResponse> {
      const encoded = encodeURIComponent(paymentId);
      return request<GetPaymentResponse>(`/api/payments/${encoded}`, {
        method: "GET",
      });
    },

    async refundPayment(paymentId: string, body: RefundRequest = {}): Promise<RefundResponse> {
      const encoded = encodeURIComponent(paymentId);
      return request<RefundResponse>(`/api/payments/${encoded}/refund`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
    },
  };
}

export type {
  CreateChargeRequest,
  CreateChargeResponse,
  TapChargeWebhookPayload,
  ChapaWebhookPayload,
  GetPaymentResponse,
  RefundRequest,
  RefundResponse,
  ErrorResponse,
  PhoneInput,
  MetadataMap,
  CustomerInput,
};
