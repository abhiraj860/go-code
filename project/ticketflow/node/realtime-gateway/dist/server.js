/**
 * realtime-gateway pushes live seat availability to browsers.
 *
 * WHY THIS EXISTS AS A SEPARATE SERVICE.
 *
 * During a drop, seat state changes thousands of times a second. Without a push
 * channel every open seat map polls, and 50k browsers polling once a second is
 * 50k requests/sec of pure overhead against inventory — for data that is
 * unchanged most of the time. One WebSocket per browser inverts that: the
 * server speaks only when something actually changed.
 *
 * It holds no state of its own and never writes. Inventory remains the single
 * authority; this is a transport.
 */
import { createServer } from 'node:http';
import { WebSocketServer } from 'ws';
import Redis from 'ioredis';
import { Hub } from './hub.js';
const PORT = Number(process.env.PORT ?? 9150);
const REDIS_URL = process.env.REDIS_URL ?? 'redis://localhost:6379';
/** Seat updates are published on this channel by inventory-svc. */
const CHANNEL_PREFIX = 'seat.updates:';
const HEARTBEAT_MS = 30_000;
const hub = new Hub();
/**
 * A dedicated Redis connection: a client in subscriber mode cannot issue normal
 * commands, so sharing one with anything else silently breaks that other thing.
 */
const subscriber = new Redis(REDIS_URL, {
    // Keep retrying rather than giving up. A gateway that stops reconnecting
    // leaves every browser silently stale with no error anywhere.
    retryStrategy: (times) => Math.min(times * 200, 5_000),
    maxRetriesPerRequest: null,
});
subscriber.on('error', (err) => {
    console.error(JSON.stringify({ level: 'error', msg: 'redis error', err: String(err) }));
});
subscriber.on('ready', () => {
    console.log(JSON.stringify({ level: 'info', msg: 'redis connected', url: REDIS_URL }));
});
// Pattern subscription, so a new event needs no resubscribe round-trip: the
// gateway is already listening for every event's channel.
await subscriber.psubscribe(`${CHANNEL_PREFIX}*`);
subscriber.on('pmessage', (_pattern, channel, message) => {
    const eventId = channel.slice(CHANNEL_PREFIX.length);
    let update;
    try {
        update = { ...JSON.parse(message), eventId };
    }
    catch {
        // A malformed publish must not take the gateway down; every other browser
        // is depending on this process staying up.
        console.error(JSON.stringify({ level: 'error', msg: 'malformed seat update', channel }));
        return;
    }
    hub.broadcast(update);
});
const httpServer = createServer((req, res) => {
    if (req.url === '/healthz') {
        res.writeHead(200, { 'content-type': 'text/plain' });
        res.end('ok');
        return;
    }
    if (req.url === '/metrics') {
        const { rooms, clients } = hub.stats();
        res.writeHead(200, { 'content-type': 'text/plain; version=0.0.4' });
        res.end(`realtime_rooms ${rooms}\nrealtime_clients ${clients}\n`);
        return;
    }
    res.writeHead(404);
    res.end();
});
const wss = new WebSocketServer({ server: httpServer, path: '/ws' });
wss.on('connection', (socket) => {
    const client = {
        send: (data) => socket.send(data),
        isAlive: true,
    };
    socket.on('pong', () => {
        client.isAlive = true;
    });
    socket.on('message', (raw) => {
        let msg;
        try {
            msg = JSON.parse(String(raw));
        }
        catch {
            socket.send(JSON.stringify({ type: 'error', message: 'malformed message' }));
            return;
        }
        if (!msg.eventId) {
            socket.send(JSON.stringify({ type: 'error', message: 'eventId is required' }));
            return;
        }
        if (msg.type === 'subscribe') {
            hub.subscribe(client, msg.eventId);
            socket.send(JSON.stringify({ type: 'subscribed', eventId: msg.eventId }));
        }
        else if (msg.type === 'unsubscribe') {
            hub.unsubscribe(client, msg.eventId);
        }
    });
    socket.on('close', () => hub.remove(client));
    socket.on('error', () => hub.remove(client));
});
/**
 * Heartbeat. A browser that closes its laptop lid leaves a socket that looks
 * open forever; without this the gateway accumulates dead connections and
 * broadcasts to sockets nobody is reading.
 */
const heartbeat = setInterval(() => {
    for (const socket of wss.clients) {
        const anySocket = socket;
        if (anySocket.isAlive === false) {
            socket.terminate();
            continue;
        }
        anySocket.isAlive = false;
        socket.ping();
    }
}, HEARTBEAT_MS);
httpServer.listen(PORT, () => {
    console.log(JSON.stringify({ level: 'info', msg: 'realtime gateway listening', port: PORT }));
});
const shutdown = async (signal) => {
    console.log(JSON.stringify({ level: 'info', msg: 'shutting down', signal }));
    clearInterval(heartbeat);
    // Close sockets before the HTTP server, so browsers see a clean close frame
    // and reconnect to another replica rather than hanging until a timeout.
    for (const socket of wss.clients)
        socket.close(1001, 'server shutting down');
    wss.close();
    httpServer.close();
    await subscriber.quit();
    process.exit(0);
};
process.on('SIGTERM', () => void shutdown('SIGTERM'));
process.on('SIGINT', () => void shutdown('SIGINT'));
//# sourceMappingURL=server.js.map