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
  title: "defer.sh | Zero-Autonomy AI",
  description:
    "AI keeps making choices you didn't ask for. Defer makes the AI ask first, then execute. Every decision tracked.",
  openGraph: {
    title: "defer.sh | Zero-Autonomy AI",
    description:
      "AI keeps making choices you didn't ask for. Defer makes the AI ask first, then execute.",
    url: "https://defer.sh",
    siteName: "defer.sh",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "defer.sh | Zero-Autonomy AI",
    description:
      "AI keeps making choices you didn't ask for. Defer makes the AI ask first, then execute.",
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
