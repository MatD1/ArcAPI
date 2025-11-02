import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'Arc Raiders API - Admin Dashboard',
  description: 'Management dashboard for Arc Raiders API',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}

