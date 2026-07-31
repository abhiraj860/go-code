'use client';

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { getSeatMap, type SeatStatus } from '@/lib/api';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import { seatToggled } from '@/store/selectionSlice';

/**
 * The interactive seat map.
 *
 * This component is the reason the event page code-splits: an arena map is tens
 * of thousands of seats plus the interaction logic, and it must not sit in the
 * bundle of a landing page most visitors never click through from. It is loaded
 * with next/dynamic and ssr:false — see the event page.
 *
 * The data split mirrors the backend exactly:
 *   - GEOMETRY comes from catalog, is effectively immutable, and is cached hard
 *   - AVAILABILITY comes from inventory, changes constantly, and is never cached
 * They arrive in one BFF call but are given very different staleTimes below.
 */
export default function SeatMap({ eventId }: { eventId: string }) {
  const dispatch = useAppDispatch();
  const selected = useAppSelector((s) => s.selection.seatIds);
  const selectionError = useAppSelector((s) => s.selection.error);

  const { data, isLoading, isError, refetch, isFetching } = useQuery({
    queryKey: ['seatmap', eventId],
    queryFn: () => getSeatMap(eventId),
    // Effectively zero. Seat state is the one thing in this system that must
    // never be served stale, so React Query is told to consider it stale the
    // instant it arrives. The realtime WebSocket pushes changes between polls;
    // this interval is the safety net for a dropped socket.
    staleTime: 0,
    refetchInterval: 15_000,
  });

  const selectedSet = useMemo(() => new Set(selected), [selected]);

  const statusBySeat = useMemo(() => {
    const map = new Map<string, SeatStatus>();
    for (const a of data?.availability ?? []) map.set(a.seatId, a.status);
    return map;
  }, [data]);

  if (isLoading) return <p className="muted">Loading seat map&hellip;</p>;
  if (isError || !data) {
    return (
      <div>
        <p className="error">Could not load the seat map.</p>
        <button className="btn" onClick={() => void refetch()}>Try again</button>
      </div>
    );
  }

  const { seat_map: map } = data;

  return (
    <div>
      <div className="row" style={{ marginBottom: 8 }}>
        <div className="muted">
          Selected {selected.length} seat{selected.length === 1 ? '' : 's'}
        </div>
        {isFetching && <span className="muted">refreshing&hellip;</span>}
      </div>

      {selectionError && <p className="error">{selectionError}</p>}

      <svg
        viewBox={`0 0 ${map.viewboxWidth} ${map.viewboxHeight}`}
        style={{ width: '100%', background: '#0e1118', borderRadius: 10 }}
        role="group"
        aria-label="Seat map"
      >
        {map.seats.map((seat) => {
          const status = statusBySeat.get(seat.id) ?? 'SEAT_STATUS_AVAILABLE';
          const isSelected = selectedSet.has(seat.id);
          // Only available seats are clickable. Held and sold seats are shown
          // rather than hidden, because an empty gap reads as a rendering bug
          // to a user, while a greyed seat reads as "taken".
          const selectable = status === 'SEAT_STATUS_AVAILABLE';

          return (
            <circle
              key={seat.id}
              cx={seat.x}
              cy={seat.y}
              r={12}
              fill={fillFor(status, isSelected)}
              stroke={isSelected ? '#fff' : 'none'}
              strokeWidth={isSelected ? 2 : 0}
              style={{ cursor: selectable ? 'pointer' : 'not-allowed' }}
              onClick={() => selectable && dispatch(seatToggled(seat.id))}
              aria-label={`Section ${seat.section} row ${seat.row} seat ${seat.number}, ${label(status)}`}
            >
              <title>{`${seat.section}-${seat.row}-${seat.number} — ${label(status)}`}</title>
            </circle>
          );
        })}
      </svg>

      <div style={{ display: 'flex', gap: 16, marginTop: 10 }} className="muted">
        <Legend colour="var(--available)" text="Available" />
        <Legend colour="var(--selected)" text="Selected" />
        <Legend colour="var(--held)" text="On hold" />
        <Legend colour="var(--sold)" text="Sold" />
      </div>
    </div>
  );
}

function Legend({ colour, text }: { colour: string; text: string }) {
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
      <span style={{ width: 10, height: 10, borderRadius: 999, background: colour }} />
      {text}
    </span>
  );
}

function fillFor(status: SeatStatus, selected: boolean): string {
  if (selected) return 'var(--selected)';
  switch (status) {
    case 'SEAT_STATUS_AVAILABLE': return 'var(--available)';
    case 'SEAT_STATUS_HELD': return 'var(--held)';
    case 'SEAT_STATUS_SOLD': return 'var(--sold)';
    default: return '#2a3040';
  }
}

function label(status: SeatStatus): string {
  switch (status) {
    case 'SEAT_STATUS_AVAILABLE': return 'available';
    case 'SEAT_STATUS_HELD': return 'on hold';
    case 'SEAT_STATUS_SOLD': return 'sold';
    default: return 'unavailable';
  }
}
