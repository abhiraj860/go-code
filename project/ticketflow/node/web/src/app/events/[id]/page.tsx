import { notFound } from 'next/navigation';
import { getEvent, listEvents, formatMoney } from '@/lib/api';
import { SeatPicker } from '@/components/SeatPicker';

/**
 * The event landing page is INCREMENTALLY STATICALLY REGENERATED.
 *
 * It is the page shared on social media and indexed by search engines, so it
 * must render fast and be crawlable — which rules out client-side rendering.
 * It is also the page a drop's traffic lands on, so per-request rendering would
 * put the whole crowd through the BFF for content that changes rarely.
 *
 * ISR gives both: served from cache, regenerated in the background every
 * `revalidate` seconds.
 *
 * The live half — seat availability — is deliberately NOT part of this render.
 * It arrives client-side via the lazy-loaded seat map, because anything the
 * server bakes into HTML is stale before the browser paints it.
 */
export const revalidate = 120;

/**
 * Prerenders the currently-listed events at build time.
 *
 * Without this the route is dynamic: Next cannot know which ids exist, so every
 * request server-renders and the `revalidate` above never applies. That is the
 * difference between claiming ISR and having it, and it shows up plainly in the
 * build output as "Dynamic" rather than "SSG".
 *
 * dynamicParams stays true (the default), so an event added after the build
 * still works -- it is rendered on first request and then cached on the same
 * revalidate interval. Prerendering covers the popular case; the tail is
 * handled lazily.
 */
export async function generateStaticParams() {
  try {
    const { events } = await listEvents(50);
    return events.map((event) => ({ id: event.id }));
  } catch {
    // The BFF may be down at build time. Returning nothing degrades this route
    // to on-demand rendering rather than failing the whole build.
    return [];
  }
}

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  try {
    const { event } = await getEvent(id);
    return {
      title: `${event.title} — TicketFlow`,
      description: `${event.title} at ${event.venue.name}, ${event.venue.city}`,
      openGraph: { title: event.title, images: event.posterUrl ? [event.posterUrl] : [] },
    };
  } catch {
    return { title: 'Event — TicketFlow' };
  }
}

export default async function EventPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;

  let data;
  try {
    data = await getEvent(id, { next: { revalidate: 120 } } as RequestInit);
  } catch {
    notFound();
  }

  const { event, content } = data;
  const summary = typeof content?.summary === 'string' ? content.summary : null;

  return (
    <>
      <h1 style={{ marginBottom: 4 }}>{event.title}</h1>
      <p className="muted">
        {event.venue.name} &middot; {event.venue.city} &middot;{' '}
        {new Date(event.startsAt).toLocaleString('en-IN', {
          weekday: 'long', day: 'numeric', month: 'long', year: 'numeric',
          hour: '2-digit', minute: '2-digit',
        })}
      </p>

      {summary && <p>{summary}</p>}

      {event.pricingTiers && event.pricingTiers.length > 0 && (
        <div className="card">
          <strong>Pricing</strong>
          {event.pricingTiers.map((tier) => (
            <div key={tier.id} className="row" style={{ marginTop: 6 }}>
              <span>{tier.name}</span>
              <span className="muted">
                {formatMoney(tier.price.amountMinor, tier.price.currencyCode)}
              </span>
            </div>
          ))}
        </div>
      )}

      <h2>Choose your seats</h2>
      <SeatPicker eventId={event.id} />
    </>
  );
}
