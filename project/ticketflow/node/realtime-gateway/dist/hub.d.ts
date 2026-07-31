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
export declare class Hub {
    /** eventId -> the sockets watching it. */
    private readonly rooms;
    /** Reverse index, so disconnecting is O(rooms joined) not O(all rooms). */
    private readonly membership;
    /** Last sequence forwarded per event, for dropping stale frames centrally. */
    private readonly lastSequence;
    subscribe(client: Client, eventId: string): void;
    unsubscribe(client: Client, eventId: string): void;
    /** Removes a client from every room it joined. Called on disconnect. */
    remove(client: Client): void;
    /**
     * Forwards an update to everyone watching, returning how many sockets it
     * reached.
     *
     * Out-of-order frames are dropped here rather than in the browser: doing it
     * once at the gateway is cheaper than making every client carry the logic,
     * and it means a buggy client cannot render a stale seat state.
     */
    broadcast(update: SeatUpdate): number;
    /** Rooms currently being watched, so the server subscribes to only those. */
    activeEvents(): string[];
    stats(): {
        rooms: number;
        clients: number;
    };
}
