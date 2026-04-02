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
    "AI keeps making choices you didn't ask for. Defer asks first, then executes. Every decision tracked.",
  openGraph: {
    title: "defer.sh | Zero-Autonomy AI",
    description:
      "AI keeps making choices you didn't ask for. Defer asks first, then executes.",
    url: "https://defer.sh",
    siteName: "defer.sh",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "defer.sh | Zero-Autonomy AI",
    description:
      "AI keeps making choices you didn't ask for. Defer asks first, then executes.",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <head>
        <script dangerouslySetInnerHTML={{ __html: `
          (function(){
            var c=document.createElement('canvas');c.width=32;c.height=32;
            var x=c.getContext('2d');
            for(var y=0;y<32;y++)for(var i=0;i<32;i++){
              x.fillStyle=Math.random()<.5?'#f97316':'#0a0a0a';
              x.fillRect(i,y,1,1);
            }
            var l=document.createElement('link');l.rel='icon';l.href=c.toDataURL();
            document.head.appendChild(l);
          })();
        `}} />
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        {children}
      </body>
    </html>
  );
}
