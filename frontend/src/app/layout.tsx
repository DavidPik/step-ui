import type { Metadata } from 'next';
import './globals.css';
import { Navigation } from './navigation';

export const metadata: Metadata = {
  title: 'step-ca Web UI',
  description: 'Web UI for step-ca service',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="app-body">
        <Navigation />
        <main className="app-main">{children}</main>
      </body>
    </html>
  );
}
