/**
 * The subscription hub: which sockets care about which events.
 *
 * Kept separate from the server and from Redis so the routing logic — the part
 * with the interesting failure modes — is unit-testable without opening a
 * socket or running a broker.
 */

/** A connected client. Structural, so tests can pass a plain object. */
export interface Client {
  send(data: string): void;
  /** Set by the server; used to drop sockets that stopped responding to pings. */
  isAlive: boolean;
}

/** A seat-state change pushed to browsers watching an event. */
export interface SeatUpdate {
  eventId: string;
  seatId: string;
  status: 'available' | 'held' | 'sold' | 'blocked';
  /**
   * Monotonic per event. Clients drop frames with a sequence lower than the
   * last one they rendered.
   *
   * Necessary because WebSocket guarantees ordering per connection but the
   * gateway has several Redis subscriptions and several replicas: two updates
   * for one seat can reach a browser out of order, and rendering the older one
   * last would show a seat as free that was just taken.
   */
  sequence: number;
  holdExpiresAt?: string;
}

export class Hub {
  /** eventId -> the sockets watching it. */
  private readonly rooms = new Map<string, Set<Client>>();
  /** Reverse index, so disconnecting is O(rooms joined) not O(all rooms). */
  private readonly membership = new Map<Client, Set<string>>();
  /**
   * Last sequence forwarded per SEAT, not per event.
   *
   * Per-event was a bug. The publisher assigns one sequence to every seat in a
   * hold, because they change at the same logical moment -- so with per-event
   * tracking the first seat set the watermark and every other seat in the same
   * hold was discarded as stale. A two-seat hold delivered one update.
   *
   * Per-seat is the correct granularity anyway: the guard exists to stop an
   * older state for a GIVEN SEAT overwriting a newer one, which says nothing
   * about seats that happen to share an event.
   */
  private readonly lastSequence = new Map<string, number>();

  subscribe(client: Client, eventId: string): void {
    let room = this.rooms.get(eventId);
    if (!room) {
      room = new Set();
      this.rooms.set(eventId, room);
    }
    room.add(client);

    let joined = this.membership.get(client);
    if (!joined) {
      joined = new Set();
      this.membership.set(client, joined);
    }
    joined.add(eventId);
  }

  unsubscribe(client: Client, eventId: string): void {
    const room = this.rooms.get(eventId);
    if (room) {
      room.delete(client);
      // Drop empty rooms, or a long-lived gateway accumulates one Set per event
      // it has ever seen — a slow leak that only shows up after weeks.
      if (room.size === 0) {
        this.rooms.delete(eventId);
        this.forgetSequences(eventId);
      }
    }
    this.membership.get(client)?.delete(eventId);
  }

  /** Removes a client from every room it joined. Called on disconnect. */
  remove(client: Client): void {
    const joined = this.membership.get(client);
    if (!joined) return;
    for (const eventId of joined) {
      const room = this.rooms.get(eventId);
      room?.delete(client);
      if (room && room.size === 0) {
        this.rooms.delete(eventId);
        this.forgetSequences(eventId);
      }
    }
    this.membership.delete(client);
  }

  /**
   * Forwards an update to everyone watching, returning how many sockets it
   * reached.
   *
   * Out-of-order frames are dropped here rather than in the browser: doing it
   * once at the gateway is cheaper than making every client carry the logic,
   * and it means a buggy client cannot render a stale seat state.
   */
  broadcast(update: SeatUpdate): number {
    const seatKey = `${update.eventId}:${update.seatId}`;
    const last = this.lastSequence.get(seatKey);
    if (last !== undefined && update.sequence <= last) {
      return 0;
    }
    this.lastSequence.set(seatKey, update.sequence);

    const room = this.rooms.get(update.eventId);
    if (!room || room.size === 0) return 0;

    const payload = JSON.stringify({ type: 'seat.update', ...update });
    let delivered = 0;

    for (const client of room) {
      try {
        client.send(payload);
        delivered++;
      } catch {
        // A socket that throws on send is already gone. Reap it rather than
        // letting one dead connection break the loop for everyone behind it.
        this.remove(client);
      }
    }
    return delivered;
  }

  /**
   * Drops every per-seat watermark for an event once nobody is watching it.
   * Without this a long-lived gateway keeps one entry per seat it has ever
   * seen -- an arena is 20k seats, so the leak is real rather than theoretical.
   */
  private forgetSequences(eventId: string): void {
    const prefix = `${eventId}:`;
    for (const key of this.lastSequence.keys()) {
      if (key.startsWith(prefix)) this.lastSequence.delete(key);
    }
  }

  /** Rooms currently being watched, so the server subscribes to only those. */
  activeEvents(): string[] {
    return [...this.rooms.keys()];
  }

  stats(): { rooms: number; clients: number } {
    return { rooms: this.rooms.size, clients: this.membership.size };
  }
}
