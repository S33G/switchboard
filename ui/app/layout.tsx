import "./globals.css";

import type { ReactNode } from "react";

export const metadata = {
  title: "Switchboard UI",
  description: "Client-only dashboard for running Docker containers.",
};

interface RootLayoutProps {
  children: ReactNode;
}

export default function RootLayout({ children }: RootLayoutProps) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-slate-950 text-slate-100">
        {children}
      </body>
    </html>
  );
}
