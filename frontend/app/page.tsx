"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { createCharge } from "@/lib/api";

export default function HomePage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [amount, setAmount] = useState("100");
  const [currency, setCurrency] = useState("ETB");
  const [firstName, setFirstName] = useState("Abebe");
  const [lastName, setLastName] = useState("Kebede");
  const [email, setEmail] = useState("abebe@example.com");
  const [countryCode, setCountryCode] = useState("251");
  const [phone, setPhone] = useState("911234567");

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const orderId = `ord_${Date.now()}`;
      const payment = await createCharge({
        orderId,
        amount: Number(amount),
        currency,
        customer: {
          firstName,
          lastName,
          email,
          phone: { countryCode, number: phone },
        },
      });

      // Persist for status page, then go to provider checkout (mock/tap/chapa).
      sessionStorage.setItem("lastPaymentId", payment.paymentId);
      if (payment.redirectUrl) {
        window.location.href = payment.redirectUrl;
        return;
      }
      router.push(`/status/${payment.paymentId}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Payment failed");
      setLoading(false);
    }
  }

  return (
    <section className="hero">
      <div className="hero-visual" aria-hidden="true" />
      <div className="hero-copy">
        <h1>BirrPay</h1>
        <p>Pay in ETB with an Ethiopia-ready backend that can run mock, Tap, or Chapa.</p>
      </div>
      <div className="hero-form">
        <form className="form-shell" onSubmit={onSubmit}>
          <h2>Start a payment</h2>
          {error ? <p className="error">{error}</p> : null}

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
                <option value="SAR">SAR</option>
              </select>
            </div>
          </div>

          <div className="grid-2">
            <div className="field">
              <label htmlFor="firstName">First name</label>
              <input id="firstName" value={firstName} onChange={(e) => setFirstName(e.target.value)} required />
            </div>
            <div className="field">
              <label htmlFor="lastName">Last name</label>
              <input id="lastName" value={lastName} onChange={(e) => setLastName(e.target.value)} />
            </div>
          </div>

          <div className="field">
            <label htmlFor="email">Email</label>
            <input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          </div>

          <div className="grid-2">
            <div className="field">
              <label htmlFor="countryCode">Country code</label>
              <input id="countryCode" value={countryCode} onChange={(e) => setCountryCode(e.target.value)} required />
            </div>
            <div className="field">
              <label htmlFor="phone">Phone</label>
              <input id="phone" value={phone} onChange={(e) => setPhone(e.target.value)} required />
            </div>
          </div>

          <button className="btn" type="submit" disabled={loading}>
            {loading ? "Creating charge…" : "Continue to checkout"}
          </button>
        </form>
      </div>
    </section>
  );
}
