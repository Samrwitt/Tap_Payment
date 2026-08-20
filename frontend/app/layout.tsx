import type { Metadata } from "next";
import { Fraunces, Manrope } from "next/font/google";
import "./globals.css";

const display = Fraunces({
  subsets: ["latin"],
  variable: "--font-display",
});

const body = Manrope({
  subsets: ["latin"],
  variable: "--font-body",
});

export const metadata: Metadata = {
  title: "BirrPay — Ethiopia-ready payments",
  description: "Demo checkout for the Tap Payment Ethiopia-ready backend",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className={`${display.variable} ${body.variable}`}>
        <div className="shell">
          <header className="topbar">
            <div className="brand">
              Birr<em>Pay</em>
            </div>
            <div className="topbar-meta">Ethiopia-ready checkout demo</div>
          </header>
          <main>{children}</main>
          <footer className="footer">Built on the monorepo Go API · mock / tap / chapa providers</footer>
        </div>
      </body>
    </html>
  );
}
