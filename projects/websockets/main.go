package main

import (
	"context"   // carries cancellation info between function calls
	"errors"    // compare error values
	"log"
	"net/http"
	"sync"      // a lock, so goroutines don't corrupt shared data
	"time"

	"github.com/coder/websocket"
)

// One connected client.
type client struct {
	// The message queue for THIS client. A channel is a pipe you
	// send values into and receive them from.
	// Buffered (size 16) so a brief slowdown doesn't block others.
	send chan []byte
	name string
}

// The hub owns every connection and does the fan-out.
type hub struct {
	// A lock protecting the map below. Each connection runs in its
	// own goroutine, so two can touch this at once.
	mu sync.Mutex
	// map[*client]bool used as a set — the bool value is ignored,
	// we only care whether the key is present.
	clients map[*client]bool
}

func newHub() *hub {
	// The & means "address of" — we build the struct and return
	// its address, so everyone shares one hub, not copies.
	return &hub{clients: make(map[*client]bool)}
}

func (h *hub) add(c *client) {
	h.mu.Lock()
	// defer means "run this when the function ends, whatever happens".
	defer h.mu.Unlock()
	h.clients[c] = true
}

func (h *hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// The two-value form tells us if it was actually there, so we
	// don't close the channel twice (which would panic).
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send) // signals the writer goroutine to stop
	}
}

// broadcast sends one message to everyone.
func (h *hub) broadcast(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for c := range h.clients {
		// select with a default never blocks. If a client's buffer
		// is full it means they're too slow — we drop the message
		// rather than let one bad connection stall everyone.
		// This is backpressure handling, and skipping it is how a
		// single slow phone freezes your whole chat server.
		select {
		case c.send <- msg:
		default:
			log.Println("dropping message for slow client", c.name)
		}
	}
}

var chat = newHub()

func main() {
	// Serve the HTML page.
	http.HandleFunc("/", servePage)
	http.HandleFunc("/ws", serveWS)

	log.Println("open http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func serveWS(w http.ResponseWriter, r *http.Request) {
	// Accept performs the upgrade handshake. After it returns, this
	// is no longer an HTTP request — it's a WebSocket.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Only pages from these origins may connect. Without this,
		// ANY website can open a socket to your server using the
		// visitor's cookies — the WebSocket version of CSRF.
		OriginPatterns: []string{"localhost:8080"},
	})
	if err != nil {
		log.Println("upgrade failed:", err)
		return
	}
	// CloseNow releases the connection if we exit unexpectedly.
	defer conn.CloseNow()

	// The name comes from the URL, e.g. /ws?name=abhiraj
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "anon"
	}

	c := &client{
		// make(chan []byte, 16) creates a pipe holding up to 16
		// messages before it blocks.
		send: make(chan []byte, 16),
		name: name,
	}

	chat.add(c)
	// Ensure cleanup happens even if the client crashes.
	defer chat.remove(c)

	chat.broadcast([]byte(name + " joined"))
	defer func() {
		chat.broadcast([]byte(name + " left"))
	}()

	// A context we can cancel to stop BOTH goroutines below at once.
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// go starts a function in the background. A goroutine is a
	// lightweight independent task Go manages for you.
	// One goroutine writes, one reads — that's the standard shape.
	go writeLoop(ctx, cancel, conn, c)

	readLoop(ctx, conn, c)
}

// readLoop handles messages coming FROM this client.
func readLoop(ctx context.Context, conn *websocket.Conn, c *client) {
	// Refuse messages over 4KB, so one client can't exhaust memory.
	conn.SetReadLimit(4096)

	for {
		// Read blocks until a message arrives or the connection dies.
		// The _ discards the message type; we treat everything as text.
		_, data, err := conn.Read(ctx)
		if err != nil {
			// Normal closure isn't an error worth logging loudly.
			// errors.Is asks "is this that specific error?" — safer
			// than == because errors are often wrapped in others.
			if !errors.Is(err, context.Canceled) {
				log.Println(c.name, "disconnected:", websocket.CloseStatus(err))
			}
			return
		}

		// Prefix with the sender's name and fan it out.
		msg := append([]byte(c.name+": "), data...)
		chat.broadcast(msg)
	}
}

// writeLoop handles messages going TO this client.
// It also sends pings, which is what detects dead connections.
func writeLoop(ctx context.Context, cancel context.CancelFunc,
	conn *websocket.Conn, c *client) {

	// cancel stops readLoop too when we return.
	defer cancel()

	// A Ticker fires on a channel at a fixed interval.
	// 30s is deliberately under the 60s most load balancers use
	// before killing an idle connection.
	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	for {
		// select waits on several channels and takes whichever
		// fires first.
		select {

		case msg, ok := <-c.send:
			// The two-value receive: ok is false once the channel
			// is closed, which is how remove() tells us to stop.
			if !ok {
				conn.Close(websocket.StatusNormalClosure, "")
				return
			}

			// Give each write its own deadline, so a stalled client
			// can't block this goroutine forever.
			wctx, wcancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(wctx, websocket.MessageText, msg)
			wcancel()

			if err != nil {
				log.Println("write failed for", c.name, err)
				return
			}

		case <-ping.C:
			// A TCP connection can be dead for MINUTES without either
			// side noticing — closing a laptop lid sends no signal.
			// Ping waits for a pong and errors if none comes back.
			pctx, pcancel := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Ping(pctx)
			pcancel()

			if err != nil {
				log.Println("ping failed, dropping", c.name)
				return
			}

		case <-ctx.Done():
			// Server shutting down or readLoop finished.
			return
		}
	}
}

// ---------------- THE PAGE ----------------

func servePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	// A raw string in backticks, so we can write HTML across many
	// lines without escaping quotes.
	w.Write([]byte(`<!doctype html>
<html>
<body style="font-family: monospace; max-width: 600px; margin: 40px auto">
<h3>chat</h3>
<div id="log" style="height:300px;overflow:auto;border:1px solid #ccc;padding:8px"></div>
<input id="msg" style="width:80%" placeholder="type and hit enter">
<div id="status" style="color:#888;margin-top:8px"></div>

<script>
const name = prompt("your name") || "anon";
let ws;
let backoff = 500;

function connect() {
  ws = new WebSocket("ws://localhost:8080/ws?name=" + encodeURIComponent(name));

  ws.onopen = () => {
    status.textContent = "connected";
    backoff = 500;              // reset the delay after a success
  };

  ws.onmessage = e => {
    const line = document.createElement("div");
    line.textContent = e.data;
    log.appendChild(line);
    log.scrollTop = log.scrollHeight;
  };

  ws.onclose = () => {
    status.textContent = "disconnected, retrying in " + backoff + "ms";
    // Reconnection is the CLIENT's job — the browser won't do it.
    // Jitter (the random part) stops every client reconnecting at
    // the same instant after a server restart and knocking it over
    // again.
    const jitter = Math.random() * 300;
    setTimeout(connect, backoff + jitter);
    backoff = Math.min(backoff * 2, 10000);   // cap at 10s
  };
}

msg.addEventListener("keydown", e => {
  if (e.key === "Enter" && msg.value && ws.readyState === WebSocket.OPEN) {
    ws.send(msg.value);
    msg.value = "";
  }
});

connect();
</script>
</body>
</html>`))
}
