/**
 * Typed access to the BFF.
 *
 * Every call goes through /api/*, which Next rewrites to the BFF. That keeps
 * the browser on a single origin, so the session cookie stays first-party and
 * checkout is not preceded by a CORS preflight on every request.
 */

export interface Venue {
  id: string;
  name: string;
  city: string;
  countryCode: string;
}

export interface Money {
  amountMinor: string;
  currencyCode: string;
}

export interface PricingTier {
  id: string;
  name: string;
  price: Money;
}

export interface TicketEvent {
  id: string;
  title: string;
  kind: string;
  status: string;
  venue: Venue;
  startsAt: string;
  seatMapId: string;
  tags?: string[];
  posterUrl?: string;
  pricingTiers?: PricingTier[];
  version: string;
}

export interface AvailabilitySummary {
  available: number;
  held: number;
  sold: number;
  blocked: number;
  total: number;
}

export interface Seat {
  id: string;
  section: string;
  row: string;
  number: string;
  pricingTierId: string;
  x: number;
  y: number;
}

export interface SeatMap {
  id: string;
  seats: Seat[];
  viewboxWidth: number;
  viewboxHeight: number;
}

export type SeatStatus =
  | 'SEAT_STATUS_AVAILABLE'
  | 'SEAT_STATUS_HELD'
  | 'SEAT_STATUS_SOLD'
  | 'SEAT_STATUS_BLOCKED';

export interface SeatAvailability {
  seatId: string;
  status: SeatStatus;
  holdExpiresAt?: string;
}

/** Format minor units for display. Never do currency maths in floats. */
export function formatMoney(minor: string | number, currency: string): string {
  const amount = typeof minor === 'string' ? Number(minor) : minor;
  const major = amount / 100;
  return new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency,
    maximumFractionDigits: 0,
  }).format(major);
}

const BASE = process.env.NEXT_PUBLIC_API_BASE ?? '/api';

/** Server-side base, used by SSG/SSR where a relative URL has no origin. */
const SERVER_BASE = process.env.BFF_URL ?? 'http://localhost:8080';

function url(path: string): string {
  return typeof window === 'undefined' ? `${SERVER_BASE}${path}` : `${BASE}${path}`;
}

async function get<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url(path), { ...init, credentials: 'include' });
  if (!res.ok) {
    throw new Error(`${path} responded ${res.status}`);
  }
  return (await res.json()) as T;
}

export interface EventListResponse {
  events: TicketEvent[];
  next_page_token: string;
}

export function listEvents(pageSize = 20, next?: RequestInit): Promise<EventListResponse> {
  return get<EventListResponse>(`/v1/events?page_size=${pageSize}`, next);
}

export interface EventDetailResponse {
  event: TicketEvent;
  availability: AvailabilitySummary | null;
  content: Record<string, unknown> | null;
}

export function getEvent(id: string, init?: RequestInit): Promise<EventDetailResponse> {
  return get<EventDetailResponse>(`/v1/events/${id}`, init);
}

export interface SeatMapResponse {
  seat_map: SeatMap;
  availability: SeatAvailability[];
  sequence: string;
}

export function getSeatMap(eventId: string): Promise<SeatMapResponse> {
  return get<SeatMapResponse>(`/v1/events/${eventId}/seatmap`);
}

export interface HoldResponse {
  hold_id: string;
  held_seat_ids: string[];
  rejected_seat_ids: string[] | null;
  expires_at: string;
}

export async function createHold(
  eventId: string,
  seatIds: string[],
  idempotencyKey: string,
): Promise<HoldResponse> {
  const res = await fetch(url('/v1/holds'), {
    method: 'POST',
    credentials: 'include',
    headers: {
      'content-type': 'application/json',
      // Required by the BFF. Generated once per checkout attempt and reused
      // across retries, so a flaky network cannot buy two sets of seats.
      'Idempotency-Key': idempotencyKey,
    },
    body: JSON.stringify({ event_id: eventId, seat_ids: seatIds, ttl_seconds: 120 }),
  });

  if (res.status === 409) {
    throw new Error('Those seats were just taken. Please pick different ones.');
  }
  if (!res.ok) {
    throw new Error(`Could not hold seats (${res.status})`);
  }
  return (await res.json()) as HoldResponse;
}
