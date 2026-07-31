import Link from 'next/link';
import { listEvents, formatMoney, type TicketEvent } from '@/lib/api';

/**
 * The browse page is STATICALLY GENERATED and revalidated on an interval (ISR).
 *
 * Why static: this is the SEO-critical page and the one a drop's traffic hits
 * first. Rendering it per request would put 50k renders/sec in front of the BFF
 * for a page whose contents change when an event is added — minutes or hours
 * apart, not milliseconds.
 *
 * Note what is NOT on this page: seat availability. A statically generated page
 * cannot carry data that changes thousands of times a second, and pretending
 * otherwise is how a cached "12 seats left" ends up wrong. Live numbers appear
 * on the event page, which fetches them client-side.
 */
export const revalidate = 60;

export default async function HomePage() {
  let events: TicketEvent[] = [];
  let failed = false;

  try {
    const data = await listEvents(20, { next: { revalidate: 60 } } as RequestInit);
    events = data.events ?? [];
  } catch {
    // At build time the BFF may not be running. A storefront that fails to
    // build because a backend was down would block every deploy.
    failed = true;
  }

  return (
    <>
      <h1>What&rsquo;s on</h1>

      {failed && (
        <p className="muted">
          Listings are temporarily unavailable. This page will refresh shortly.
        </p>
      )}

      {!failed && events.length === 0 && <p className="muted">No events on sale right now.</p>}

      {events.map((event) => {
        const cheapest = event.pricingTiers?.[0];
        return (
          <article key={event.id} className="card">
            <Link href={`/events/${event.id}`}>
              <div className="row">
                <div>
                  <h2 style={{ margin: '0 0 4px' }}>{event.title}</h2>
                  <div className="muted">
                    {event.venue.name} &middot; {event.venue.city} &middot;{' '}
                    {new Date(event.startsAt).toLocaleDateString('en-IN', {
                      weekday: 'short', day: 'numeric', month: 'short', year: 'numeric',
                    })}
                  </div>
                </div>
                {cheapest && (
                  <div className="muted">
                    from {formatMoney(cheapest.price.amountMinor, cheapest.price.currencyCode)}
                  </div>
                )}
              </div>
              {event.tags && event.tags.length > 0 && (
                <div style={{ marginTop: 8, display: 'flex', gap: 6 }}>
                  {event.tags.map((tag) => (
                    <span key={tag} className="pill">{tag}</span>
                  ))}
                </div>
              )}
            </Link>
          </article>
        );
      })}
    </>
  );
}
