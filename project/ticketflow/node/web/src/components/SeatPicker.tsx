'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import dynamic from 'next/dynamic';
import { useRouter } from 'next/navigation';
import { createHold } from '@/lib/api';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import { eventOpened, holdAcquired, holdFailed, seatTakenElsewhere } from '@/store/selectionSlice';

/**
 * LAZY LOADED. The seat map is the heaviest component in the app — an arena is
 * tens of thousands of SVG nodes — and most visitors to an event page never
 * open it. Splitting it out keeps it off the critical path.
 *
 * ssr: false because it renders live availability, which by definition cannot
 * be server-rendered: any HTML the server produces is stale before it reaches
 * the browser, and would flash to different values on hydration.
 */
const SeatMap = dynamic(() => import('./SeatMap'), {
  ssr: false,
  loading: () => <p className="muted">Loading seat map&hellip;</p>,
});

export function SeatPicker({ eventId }: { eventId: string }) {
  const dispatch = useAppDispatch();
  const router = useRouter();
  const { seatIds, error, holdExpiresAt } = useAppSelector((s) => s.selection);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    dispatch(eventOpened(eventId));
  }, [dispatch, eventId]);

  // One idempotency key per checkout attempt, generated once and reused across
  // retries. Generating a fresh key per request would make every retry a new
  // purchase — the exact bug the header exists to prevent.
  const idempotencyKey = useMemo(
    () => (typeof crypto !== 'undefined' ? crypto.randomUUID() : String(Date.now())),
    [],
  );

  useRealtimeSeatUpdates(eventId, (seatId) => dispatch(seatTakenElsewhere(seatId)));

  const hold = useCallback(async () => {
    if (seatIds.length === 0) return;
    setSubmitting(true);
    try {
      const res = await createHold(eventId, seatIds, idempotencyKey);
      dispatch(
        holdAcquired({
          holdId: res.hold_id,
          seatIds: res.held_seat_ids,
          expiresAt: res.expires_at,
        }),
      );
      router.push('/checkout');
    } catch (err) {
      dispatch(holdFailed(err instanceof Error ? err.message : 'Could not hold those seats.'));
    } finally {
      setSubmitting(false);
    }
  }, [dispatch, eventId, idempotencyKey, router, seatIds]);

  return (
    <section>
      <SeatMap eventId={eventId} />

      {error && <p className="error">{error}</p>}
      {holdExpiresAt && <HoldCountdown expiresAt={holdExpiresAt} />}

      <button className="btn" disabled={seatIds.length === 0 || submitting} onClick={() => void hold()}>
        {submitting
          ? 'Holding seats…'
          : seatIds.length === 0
            ? 'Select seats to continue'
            : `Hold ${seatIds.length} seat${seatIds.length === 1 ? '' : 's'}`}
      </button>
    </section>
  );
}

/** Shows how long the buyer has left, driven by the server's expiry. */
function HoldCountdown({ expiresAt }: { expiresAt: string }) {
  const [remaining, setRemaining] = useState(() => secondsUntil(expiresAt));

  useEffect(() => {
    const id = setInterval(() => setRemaining(secondsUntil(expiresAt)), 1000);
    return () => clearInterval(id);
  }, [expiresAt]);

  if (remaining <= 0) return <p className="error">Your hold expired. Please pick seats again.</p>;

  const mins = Math.floor(remaining / 60);
  const secs = String(remaining % 60).padStart(2, '0');
  return <p className="muted">Seats held for {mins}:{secs}</p>;
}

function secondsUntil(iso: string): number {
  return Math.max(0, Math.floor((new Date(iso).getTime() - Date.now()) / 1000));
}

/**
 * Subscribes to live seat changes.
 *
 * The point is not to render other people's seats moving — it is to notice
 * immediately when a seat THIS user selected is taken, so they find out while
 * still on the map rather than at checkout.
 */
function useRealtimeSeatUpdates(eventId: string, onSeatTaken: (seatId: string) => void) {
  useEffect(() => {
    const base = process.env.NEXT_PUBLIC_WS_URL ?? 'ws://localhost:9150/ws';
    let socket: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
    let closed = false;

    const connect = () => {
      if (closed) return;
      socket = new WebSocket(base);

      socket.onopen = () => socket?.send(JSON.stringify({ type: 'subscribe', eventId }));

      socket.onmessage = (event) => {
        try {
          const msg = JSON.parse(String(event.data)) as {
            type?: string; seatId?: string; status?: string;
          };
          if (msg.type === 'seat.update' && msg.seatId && msg.status !== 'available') {
            onSeatTaken(msg.seatId);
          }
        } catch {
          // A malformed frame must not break the socket for every later one.
        }
      };

      // Reconnect with a fixed short delay. The seat map polls every 15s as a
      // backstop, so a missed reconnect degrades to polling rather than to
      // stale data.
      socket.onclose = () => {
        if (!closed) reconnectTimer = setTimeout(connect, 2000);
      };
    };

    connect();

    return () => {
      closed = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      socket?.close();
    };
  }, [eventId, onSeatTaken]);
}
