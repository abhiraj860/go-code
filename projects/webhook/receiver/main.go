package main

import (
	"crypto/hmac"   // create a signature using a shared secret
	"crypto/sha256" // the hashing algorithm used for that signature
	"encoding/hex"  // turn raw bytes into readable hex text
	"encoding/json" // convert JSON text <-> Go structs
	"fmt"           // build formatted strings
	"io"            // read a stream of bytes
	"log"           // print messages
	"net/http"      // build the web server
	"os"            // read environment variables
	"time"          // timestamps and durations
)

// The shared secret. Both sender and receiver must know the same value —
// that's what proves a request really came from the sender.
var secret = os.Getenv("WEBHOOK_SECRET")

// The shape of the data we expect. The backtick parts are "tags"
// telling the JSON decoder which JSON key maps to which field.
type Payload struct {
	Event string `json:"event"`
	ID    string `json:"id"`
	Data  struct {
		OrderID int    `json:"order_id"`
		Status  string `json:"status"`
	} `json:"data"`
}

// Remembers which webhook IDs we've already handled.
// map[string]bool means "keys are strings, values are true/false".
// Senders retry, so the same webhook can arrive twice — this stops
// us from processing it twice.
var seen = map[string]bool{}

func main() {
	// Refuse to start without a secret — an unsigned webhook endpoint
	// is one anybody on the internet can post to.
	if secret == "" {
		log.Fatal("set WEBHOOK_SECRET")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhook", handleWebhook)

	log.Println("receiver listening on :9000")
	// Blocks forever; only returns if the server crashes.
	log.Fatal(http.ListenAndServe(":9000", mux))
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Read the WHOLE body into memory first. We need the exact raw
	// bytes to check the signature — parsing it first would change
	// the spacing and the signature would no longer match.
	// MaxBytesReader caps the size so a huge request can't exhaust
	// our memory. 1<<20 means 1 shifted left 20 places = 1048576 = 1MB.
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "cannot read body", 400)
		return
	}
	// defer means "run this when the function ends, whatever happens".
	defer r.Body.Close()

	// Pull the sender's signature and timestamp out of the headers.
	// Header.Get is case-insensitive, so capitalisation doesn't matter.
	sig := r.Header.Get("X-Signature")
	ts := r.Header.Get("X-Timestamp")

	// Reject anything unsigned before doing any other work.
	if !verify(body, ts, sig) {
		log.Println("REJECTED: bad signature")
		http.Error(w, "bad signature", 401) // 401 = not authorised
		return
	}

	var p Payload
	// Unmarshal parses the JSON bytes into the struct.
	// The & is required because Unmarshal WRITES INTO p, so it needs
	// p's memory address rather than a copy of it.
	if err := json.Unmarshal(body, &p); err != nil {
		http.Error(w, "bad json", 400)
		return
	}

	// If we've handled this ID before, say "fine" and do nothing.
	// This is called being idempotent — repeating it changes nothing.
	if seen[p.ID] {
		log.Println("duplicate, ignoring:", p.ID)
		w.WriteHeader(200)
		return
	}
	seen[p.ID] = true

	// Reply FAST, before doing the real work. Senders usually give you
	// only a few seconds before treating it as a failure and retrying.
	w.WriteHeader(200)

	// go starts this function in the background so it runs while the
	// reply is already on its way. A goroutine is a lightweight
	// independent task Go manages for you.
	go process(p)
}

// The slow work, done after we've already replied.
func process(p Payload) {
	log.Printf("processing %s: order %d -> %s", p.Event, p.Data.OrderID, p.Data.Status)
	time.Sleep(2 * time.Second) // pretend this is a database write
	log.Println("done:", p.ID)
}

// verify recomputes the signature and compares it to the sender's.
func verify(body []byte, ts, sig string) bool {
	// Missing pieces mean it can't possibly be valid.
	if sig == "" || ts == "" {
		return false
	}

	// Sign the timestamp AND body together. Including the timestamp
	// means an attacker can't take an old valid request and resend it
	// later — the timestamp would be stale.
	// Sprintf builds a string; %s inserts a value.
	signed := fmt.Sprintf("%s.%s", ts, string(body))

	// hmac.New starts a signature calculation using our secret.
	// []byte(secret) converts the string into raw bytes.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed)) // feed the data in
	// Sum(nil) finishes and returns the raw signature bytes;
	// EncodeToString turns those into readable hex characters.
	want := hex.EncodeToString(mac.Sum(nil))

	// hmac.Equal instead of == because it always takes the same amount
	// of time regardless of where the strings differ. A normal
	// comparison stops at the first mismatch, and timing that leak
	// lets an attacker guess the signature character by character.
	return hmac.Equal([]byte(sig), []byte(want))
}