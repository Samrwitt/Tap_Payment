"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  customerKeyFromPhone,
  listPaymentMethods,
  oneTapPay,
  savePaymentMethod,
  type PaymentMethod,
} from "@/lib/api";

const STORAGE_KEY = "birrapay_customer";

type StoredCustomer = {
  customerKey: string;
  paymentMethodId: string;
  label: string;
  firstName: string;
  countryCode: string;
  phone: string;
};

export default function HomePage() {
  const router = useRouter();
  const [stored, setStored] = useState<StoredCustomer | null>(null);
  const [ready, setReady] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [amount, setAmount] = useState("100");
  const [currency, setCurrency] = useState("ETB");

  // setup fields
  const [firstName, setFirstName] = useState("Abebe");
  const [email, setEmail] = useState("abebe@example.com");
  const [countryCode, setCountryCode] = useState("251");
  const [phone, setPhone] = useState("911234567");
  const [methodType, setMethodType] = useState<"wallet" | "card">("wallet");
  const [cardNumber, setCardNumber] = useState("4111111111111111");

  useEffect(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) setStored(JSON.parse(raw) as StoredCustomer);
    } catch {
      // ignore
    }
    setReady(true);
  }, []);

  const headline = useMemo(() => {
    if (stored) return "One tap.";
    return "Set up once.";
  }, [stored]);

  async function onSetup(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const key = customerKeyFromPhone(countryCode, phone);
      const method = await savePaymentMethod({
        customerKey: key,
        methodType,
        cardNumber: methodType === "card" ? cardNumber : undefined,
        customer: {
          firstName,
          email,
          phone: { countryCode, number: phone },
        },
      });
      const next: StoredCustomer = {
        customerKey: key,
        paymentMethodId: method.id,
        label: method.label,
        firstName,
        countryCode,
        phone,
      };
      localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
      setStored(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Setup failed");
    } finally {
      setLoading(false);
    }
  }

  async function onOneTap(e: FormEvent) {
    e.preventDefault();
    if (!stored) return;
    setError(null);
    setLoading(true);
    try {
      // refresh method list in case backend restarted with new DB
      const listed = await listPaymentMethods(stored.customerKey);
      let method: PaymentMethod | undefined = listed.methods.find((m) => m.id === stored.paymentMethodId);
      if (!method && listed.methods[0]) method = listed.methods[0];
      if (!method) {
        localStorage.removeItem(STORAGE_KEY);
        setStored(null);
        throw new Error("Saved method missing. Set up one-tap again.");
      }

      const payment = await oneTapPay({
        orderId: `ord_${Date.now()}`,
        amount: Number(amount),
        currency,
        paymentMethodId: method.id,
      });
      router.push(`/status/${payment.paymentId}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "One-tap failed");
      setLoading(false);
    }
  }

  function resetMethod() {
    localStorage.removeItem(STORAGE_KEY);
    setStored(null);
    setError(null);
  }

  if (!ready) return null;

  return (
    <section className="hero">
      <div className="hero-visual" aria-hidden="true" />
      <div className="hero-copy">
        <h1>BirrPay</h1>
        <p>
          {stored
            ? "Pay with one tap using your saved Ethiopian wallet or card."
            : "Save a payment method once, then pay with a single tap."}
        </p>
      </div>
      <div className="hero-form">
        <div className="form-shell">
          <h2>{headline}</h2>
          {error ? <p className="error">{error}</p> : null}

          {stored ? (
            <form onSubmit={onOneTap}>
              <p style={{ margin: "0 0 1rem", color: "var(--muted)" }}>
                Paying as <strong>{stored.firstName}</strong> · {stored.label}
              </p>
              <div className="grid-2">
                <div className="field">
                  <label htmlFor="amount">Amount</label>
                  <input
                    id="amount"
                    type="number"
                    min="1"
                    step="0.01"
                    value={amount}
                    onChange={(e) => setAmount(e.target.value)}
                    required
                  />
                </div>
                <div className="field">
                  <label htmlFor="currency">Currency</label>
                  <select id="currency" value={currency} onChange={(e) => setCurrency(e.target.value)}>
                    <option value="ETB">ETB</option>
                    <option value="USD">USD</option>
                  </select>
                </div>
              </div>
              <button className="btn" type="submit" disabled={loading}>
                {loading ? "Paying…" : `Pay ${amount} ${currency} · one tap`}
              </button>
              <button
                type="button"
                onClick={resetMethod}
                style={{
                  marginTop: "0.75rem",
                  width: "100%",
                  background: "transparent",
                  border: "1px solid var(--line)",
                  borderRadius: "0.55rem",
                  padding: "0.7rem",
                  cursor: "pointer",
                }}
              >
                Change payment method
              </button>
            </form>
          ) : (
            <form onSubmit={onSetup}>
              <div className="field">
                <label htmlFor="methodType">Method</label>
                <select
                  id="methodType"
                  value={methodType}
                  onChange={(e) => setMethodType(e.target.value as "wallet" | "card")}
                >
                  <option value="wallet">Phone wallet (Telebirr-style)</option>
                  <option value="card">Card</option>
                </select>
              </div>
              <div className="field">
                <label htmlFor="firstName">First name</label>
                <input id="firstName" value={firstName} onChange={(e) => setFirstName(e.target.value)} required />
              </div>
              <div className="field">
                <label htmlFor="email">Email</label>
                <input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
              </div>
              <div className="grid-2">
                <div className="field">
                  <label htmlFor="countryCode">Code</label>
                  <input id="countryCode" value={countryCode} onChange={(e) => setCountryCode(e.target.value)} required />
                </div>
                <div className="field">
                  <label htmlFor="phone">Phone</label>
                  <input id="phone" value={phone} onChange={(e) => setPhone(e.target.value)} required />
                </div>
              </div>
              {methodType === "card" ? (
                <div className="field">
                  <label htmlFor="cardNumber">Card number (demo)</label>
                  <input id="cardNumber" value={cardNumber} onChange={(e) => setCardNumber(e.target.value)} required />
                </div>
              ) : null}
              <button className="btn" type="submit" disabled={loading}>
                {loading ? "Saving…" : "Enable one-tap"}
              </button>
            </form>
          )}
        </div>
      </div>
    </section>
  );
}
