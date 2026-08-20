"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { getPayment, type PaymentStatus } from "@/lib/api";

export default function StatusPage({ params }: { params: Promise<{ paymentId: string }> }) {
  const { paymentId } = use(params);
  const [payment, setPayment] = useState<PaymentStatus | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setInterval> | undefined;

    async function load() {
      try {
        const data = await getPayment(paymentId);
        if (!cancelled) {
          setPayment(data);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load payment");
        }
      }
    }

    load();
    timer = setInterval(load, 2500);
    return () => {
      cancelled = true;
      if (timer) clearInterval(timer);
    };
  }, [paymentId]);

  const status = (payment?.status || "").toLowerCase();
  const ok = status === "captured" || status === "paid" || status === "success" || status === "successful";

  return (
    <section className="status-page">
      <h1>Payment status</h1>
      <p>
        Live status for <code>{paymentId}</code>
      </p>

      {error ? <p className="error">{error}</p> : null}

      {payment ? (
        <>
          <div className={`badge ${ok ? "" : "bad"}`}>{payment.status}</div>
          <div className="meta-list">
            <div>
              <span>Provider</span>
              <strong>{payment.provider}</strong>
            </div>
            <div>
              <span>Provider charge</span>
              <strong>{payment.providerPaymentId || "—"}</strong>
            </div>
            <div>
              <span>Order</span>
              <strong>{payment.orderId}</strong>
            </div>
            <div>
              <span>Updated</span>
              <strong>{new Date(payment.updatedAt).toLocaleString()}</strong>
            </div>
          </div>
          {payment.redirectUrl ? (
            <p style={{ marginTop: "1.25rem" }}>
              <a className="btn" href={payment.redirectUrl} style={{ display: "inline-block", width: "auto" }}>
                Open checkout again
              </a>
            </p>
          ) : null}
        </>
      ) : (
        <p style={{ marginTop: "1rem", color: "var(--muted)" }}>Loading…</p>
      )}

      <p style={{ marginTop: "1.75rem" }}>
        <Link href="/">← New payment</Link>
      </p>
    </section>
  );
}
