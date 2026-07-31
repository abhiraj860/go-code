import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Hub, type Client, type SeatUpdate } from './hub.js';

/** A recording stand-in for a WebSocket. */
function fakeClient(): Client & { sent: string[] } {
  const sent: string[] = [];
  return {
    sent,
    isAlive: true,
    send(data: string) {
      sent.push(data);
    },
  };
}

function update(overrides: Partial<SeatUpdate> = {}): SeatUpdate {
  return {
    eventId: 'evt-1',
    seatId: 'S-1',
    status: 'held',
    sequence: 1,
    ...overrides,
  };
}

test('delivers only to clients watching the event', () => {
  const hub = new Hub();
  const watching = fakeClient();
  const elsewhere = fakeClient();

  hub.subscribe(watching, 'evt-1');
  hub.subscribe(elsewhere, 'evt-2');

  const delivered = hub.broadcast(update());

  assert.equal(delivered, 1);
  assert.equal(watching.sent.length, 1);
  assert.equal(elsewhere.sent.length, 0, 'a client watching another event received an update');

  const payload = JSON.parse(watching.sent[0]!);
  assert.equal(payload.type, 'seat.update');
  assert.equal(payload.seatId, 'S-1');
});

test('fans out to every client in a room', () => {
  const hub = new Hub();
  const clients = [fakeClient(), fakeClient(), fakeClient()];
  for (const c of clients) hub.subscribe(c, 'evt-1');

  assert.equal(hub.broadcast(update()), 3);
  for (const c of clients) assert.equal(c.sent.length, 1);
});

// The reason sequence exists: WebSocket preserves order per connection, but the
// gateway has several Redis subscriptions and several replicas, so two updates
// for one seat can arrive out of order. Rendering the older one last would show
// a seat as free that was just taken.
test('drops out-of-order and duplicate frames', () => {
  const hub = new Hub();
  const client = fakeClient();
  hub.subscribe(client, 'evt-1');

  assert.equal(hub.broadcast(update({ sequence: 5, status: 'held' })), 1);
  assert.equal(hub.broadcast(update({ sequence: 3, status: 'available' })), 0, 'a stale frame was delivered');
  assert.equal(hub.broadcast(update({ sequence: 5, status: 'available' })), 0, 'a duplicate frame was delivered');
  assert.equal(hub.broadcast(update({ sequence: 6, status: 'sold' })), 1);

  assert.equal(client.sent.length, 2);
  assert.equal(JSON.parse(client.sent[1]!).status, 'sold');
});

test('unsubscribe stops delivery', () => {
  const hub = new Hub();
  const client = fakeClient();

  hub.subscribe(client, 'evt-1');
  hub.unsubscribe(client, 'evt-1');

  assert.equal(hub.broadcast(update()), 0);
  assert.equal(client.sent.length, 0);
});

test('remove detaches a client from every room it joined', () => {
  const hub = new Hub();
  const client = fakeClient();

  hub.subscribe(client, 'evt-1');
  hub.subscribe(client, 'evt-2');
  assert.equal(hub.stats().clients, 1);

  hub.remove(client);

  assert.equal(hub.broadcast(update({ eventId: 'evt-1' })), 0);
  assert.equal(hub.broadcast(update({ eventId: 'evt-2' })), 0);
  assert.deepEqual(hub.stats(), { rooms: 0, clients: 0 });
});

// A long-lived gateway that keeps one empty Set per event it has ever seen
// leaks slowly, and only becomes visible after weeks of uptime.
test('empty rooms are reclaimed', () => {
  const hub = new Hub();
  const client = fakeClient();

  for (let i = 0; i < 100; i++) hub.subscribe(client, `evt-${i}`);
  assert.equal(hub.stats().rooms, 100);

  hub.remove(client);
  assert.equal(hub.stats().rooms, 0, 'rooms leaked after the last client left');
});

// One dead socket must not stop delivery to everyone behind it in the loop.
test('a throwing socket is reaped without breaking the broadcast', () => {
  const hub = new Hub();
  const healthy = fakeClient();
  const broken: Client = {
    isAlive: true,
    send() {
      throw new Error('socket closed');
    },
  };

  hub.subscribe(broken, 'evt-1');
  hub.subscribe(healthy, 'evt-1');

  const delivered = hub.broadcast(update({ sequence: 1 }));

  assert.equal(delivered, 1, 'the healthy client did not receive the update');
  assert.equal(healthy.sent.length, 1);
  // And the broken one is gone, so it is not retried forever.
  assert.equal(hub.stats().clients, 1);
});

test('subscribing twice does not duplicate delivery', () => {
  const hub = new Hub();
  const client = fakeClient();

  hub.subscribe(client, 'evt-1');
  hub.subscribe(client, 'evt-1');

  assert.equal(hub.broadcast(update()), 1);
  assert.equal(client.sent.length, 1);
});

test('sequence tracking is per event, not global', () => {
  const hub = new Hub();
  const client = fakeClient();
  hub.subscribe(client, 'evt-1');
  hub.subscribe(client, 'evt-2');

  assert.equal(hub.broadcast(update({ eventId: 'evt-1', sequence: 10 })), 1);
  // A low sequence on a DIFFERENT event must still be delivered; sharing one
  // counter across events would silently drop updates for quieter events.
  assert.equal(hub.broadcast(update({ eventId: 'evt-2', sequence: 1 })), 1);
  assert.equal(client.sent.length, 2);
});

test('broadcast to an empty room is a no-op', () => {
  const hub = new Hub();
  assert.equal(hub.broadcast(update()), 0);
});

// Regression test for a real bug. The publisher gives every seat in one hold
// the same sequence, because they change at the same logical moment. With a
// per-EVENT watermark the first seat set the bar and every other seat in that
// hold was discarded as stale, so a two-seat hold delivered one update and the
// browser showed one seat still free.
test('seats sharing a sequence are all delivered', () => {
  const hub = new Hub();
  const client = fakeClient();
  hub.subscribe(client, 'evt-1');

  // Exactly what a two-seat hold publishes.
  assert.equal(hub.broadcast(update({ seatId: 'A-1', status: 'held', sequence: 1 })), 1);
  assert.equal(hub.broadcast(update({ seatId: 'A-2', status: 'held', sequence: 1 })), 1);

  assert.equal(client.sent.length, 2, 'a seat sharing a sequence with another was dropped');
  assert.deepEqual(
    client.sent.map((s) => JSON.parse(s).seatId).sort(),
    ['A-1', 'A-2'],
  );
});

test('the stale guard still applies per seat', () => {
  const hub = new Hub();
  const client = fakeClient();
  hub.subscribe(client, 'evt-1');

  assert.equal(hub.broadcast(update({ seatId: 'A-1', status: 'sold', sequence: 5 })), 1);
  // Older state for the SAME seat is still dropped.
  assert.equal(hub.broadcast(update({ seatId: 'A-1', status: 'available', sequence: 4 })), 0);
  // But a different seat at a lower sequence is unaffected.
  assert.equal(hub.broadcast(update({ seatId: 'A-2', status: 'held', sequence: 2 })), 1);
});

test('per-seat watermarks are reclaimed with the room', () => {
  const hub = new Hub();
  const client = fakeClient();
  hub.subscribe(client, 'evt-1');
  for (let i = 0; i < 500; i++) {
    hub.broadcast(update({ seatId: `S-${i}`, sequence: 1 }));
  }
  hub.remove(client);
  assert.deepEqual(hub.stats(), { rooms: 0, clients: 0 });
});
