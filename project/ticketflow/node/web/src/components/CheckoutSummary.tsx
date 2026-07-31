'use client';

import Link from 'next/link';
import { useAppSelector } from '@/store/hooks';

/**
 * Reads the hold from Redux rather than refetching it.
 *
 * This is the payoff for keeping selection in client state: the buyer arrives
 * from the seat map with their hold already in hand, so checkout renders
 * instantly with no round-trip and no spinner at the most abandonment-prone
 * moment in the funnel.
 */
export function CheckoutSummary() {
  const { seatIds, holdId, holdExpiresAt, eventId } = useAppSelector((s) => s.selection);

  if (!holdId || seatIds.length === 0) {
    return (
      <div className="card">
        <p>You don&rsquo;t have any seats on hold.</p>
        <Link className="btn" href="/" style={{ display: 'inline-block', textDecoration: 'none' }}>
          Browse events
        </Link>
      </div>
    );
  }

  const expired = holdExpiresAt ? new Date(holdExpiresAt).getTime() < Date.now() : false;

  return (
    <div className="card">
      <div className="row">
        <strong>{seatIds.length} seat{seatIds.length === 1 ? '' : 's'} held</strong>
        <span className="muted">hold {holdId.slice(0, 8)}</span>
      </div>

      <ul style={{ margin: '10px 0', paddingLeft: 18 }}>
        {seatIds.map((id) => <li key={id}>{id}</li>)}
      </ul>

      {expired ? (
        <>
          <p className="error">This hold has expired and the seats have been released.</p>
          {eventId && <Link className="btn" href={`/events/${eventId}`}>Pick seats again</Link>}
        </>
      ) : (
        <>
          <p className="muted">
            Payment is wired in Phase 4, behind the API Gateway idempotency layer.
          </p>
          <button className="btn" disabled>Pay (coming in Phase 4)</button>
        </>
      )}
    </div>
  );
}
