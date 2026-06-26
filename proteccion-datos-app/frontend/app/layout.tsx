import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Protección de Datos — Autodiagnóstico",
  description: "Plataforma SaaS de autodiagnóstico de cumplimiento Ley 1581 / GDPR",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="es">
      <body className="min-h-screen antialiased">{children}</body>
    </html>
  );
}
