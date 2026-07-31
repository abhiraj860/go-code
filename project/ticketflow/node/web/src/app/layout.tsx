import type { Metadata } from 'next';
import { Providers } from './providers';
import './globals.css';

export const metadata: Metadata = {
  title: 'TicketFlow',
  description: 'Live event ticketing',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <Providers>
          <header className="site-header">
            <a href="/" className="brand">TicketFlow</a>
          </header>
          <main className="container">{children}</main>
        </Providers>
      </body>
    </html>
  );
}
