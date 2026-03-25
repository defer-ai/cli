import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Defer — Zero-Autonomy AI",
  description:
    "Every decision is yours. Defer is a design philosophy where AI makes zero decisions that belong to the human.",
  openGraph: {
    title: "Defer — Zero-Autonomy AI",
    description:
      "Every decision is yours. The AI's job is to find every decision hidden in a task, surface it, and wait.",
    url: "https://defer.sh",
    siteName: "Defer",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "Defer — Zero-Autonomy AI",
    description:
      "Every decision is yours. The AI's job is to find every decision hidden in a task, surface it, and wait.",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        {children}
      </body>
    </html>
  );
}
