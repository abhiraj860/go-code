import { CheckoutSummary } from '@/components/CheckoutSummary';

/**
 * Checkout is SERVER-RENDERED per request and never cached.
 *
 * The opposite call from the landing page, for the opposite reason: this page
 * is personal to one buyer and reflects a hold that expires in minutes. A
 * cached checkout page shown to a second user would leak the first user's
 * selection, and a stale one would show a hold that has already lapsed.
 */
export const dynamic = 'force-dynamic';

export const metadata = {
  title: 'Checkout — TicketFlow',
  // Never index a page that only makes sense for one signed-in buyer.
  robots: { index: false, follow: false },
};

export default function CheckoutPage() {
  return (
    <>
      <h1>Checkout</h1>
      <CheckoutSummary />
    </>
  );
}
