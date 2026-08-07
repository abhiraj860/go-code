package main

import (
	"context"       // carries cancellation info between function calls
	"crypto/rand"   // cryptographically secure randomness — NOT math/rand
	"crypto/sha256" // hashing, so we never store the raw token
	"crypto/subtle" // comparisons that take constant time
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// The cookie's name. The __Host- prefix is a browser-enforced
	// rule: the cookie MUST be Secure, path=/, and have no Domain.
	// It stops a subdomain from overwriting your session cookie.
	// Requires HTTPS, so we switch it off in dev below.
	cookieNameSecure = "__Host-session"
	cookieNameDev    = "session"

	// Two separate clocks. Idle expires after inactivity; absolute
	// expires no matter how active you are, so a stolen token can't
	// live forever by being used constantly.
	idleTimeout     = 30 * time.Minute
	absoluteTimeout = 12 * time.Hour

	// How many random bytes in a session ID. 32 bytes = 256 bits,
	// far beyond guessable.
	tokenBytes = 32
)

// What we store about each session. The backtick parts are "tags"
// telling the JSON encoder what to name each field.
type Session struct {
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	// Recording these lets you show "active devices" and spot
	// a session suddenly jumping continents.
	UserAgent string `json:"user_agent"`
	IP        string `json:"ip"`
}

// Shared handles. The * means "pointer to" — holds the address of the
// real object rather than a copy.
var (
	rdb    *redis.Client
	devMode bool // true when running without HTTPS locally
)

func main() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	// devMode relaxes the cookie rules that require HTTPS.
	// NEVER set this in production — it disables Secure.
	devMode = os.Getenv("DEV_MODE") == "true"
	if devMode {
		log.Println("WARNING: dev mode — cookies are not Secure")
	}

	// The & means "address of": we build an Options struct and pass
	// its address, which is what the library expects.
	rdb = redis.NewClient(&redis.Options{Addr: addr})

	// context.Background() is an empty, do-nothing context — a
	// required argument that carries "stop now" signals.
	// Ping proves Redis is really reachable before we serve traffic.
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("cannot reach redis: ", err) // print and quit
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", handleLogin)
	mux.HandleFunc("POST /logout", handleLogout)
	mux.HandleFunc("POST /logout-all", requireAuth(handleLogoutAll))
	mux.HandleFunc("GET /me", requireAuth(handleMe))
	mux.HandleFunc("GET /sessions", requireAuth(handleListSessions))

	// An explicit Server so we can set timeouts. The default
	// ListenAndServe has none, meaning a slow client can hold a
	// connection open forever.
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("listening on :8080")
	log.Fatal(srv.ListenAndServe())
}

// ---------------- TOKEN HELPERS ----------------

// newToken makes an unguessable session ID.
func newToken() (string, error) {
	b := make([]byte, tokenBytes) // an empty list of 32 bytes

	// rand.Read fills it with secure random bytes. This is
	// crypto/rand — math/rand is predictable and would be a
	// full account-takeover bug here.
	if _, err := rand.Read(b); err != nil {
		return "", err // _ discards the byte count we don't need
	}

	// Turn the raw bytes into URL-safe text for the cookie.
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken turns the token into a fingerprint for storage.
// We store the hash, never the token itself — so if Redis is dumped,
// the attacker gets hashes they can't use as session IDs.
func hashToken(t string) string {
	// Sum256 hashes the bytes; [] around it because it returns an
	// array, and hex.EncodeToString wants a slice.
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:]) // sum[:] converts array to slice
}

// Redis key names. A prefix keeps different data types apart.
func sessionKey(hash string) string { return "session:" + hash }
func userKey(uid string) string     { return "user:" + uid + ":sessions" }

// ---------------- SESSION STORE ----------------

// createSession writes a new session to Redis and returns the token.
func createSession(ctx context.Context, s Session) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	h := hashToken(token)

	// Marshal turns the struct into JSON bytes for storage.
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}

	// A pipeline sends several commands in one network round trip
	// instead of one at a time.
	pipe := rdb.TxPipeline()

	// Set with a TTL (time to live). Redis deletes the key itself
	// when the clock runs out — no cleanup job needed, and this IS
	// our idle timeout.
	pipe.Set(ctx, sessionKey(h), data, idleTimeout)

	// Also track which sessions belong to this user, so "log out
	// everywhere" doesn't need a slow scan of every key.
	// SAdd adds to a set — a collection with no duplicates.
	pipe.SAdd(ctx, userKey(s.UserID), h)
	// The set needs its own expiry or it grows forever.
	pipe.Expire(ctx, userKey(s.UserID), absoluteTimeout)

	// Exec runs everything. The _ discards the per-command results.
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}

	return token, nil
}

// A named error so callers can check for it specifically.
var errNoSession = errors.New("no valid session")

// getSession looks up a session and refreshes its idle clock.
func getSession(ctx context.Context, token string) (*Session, string, error) {
	h := hashToken(token)

	// Get returns the value and an error.
	data, err := rdb.Get(ctx, sessionKey(h)).Result()
	if err != nil {
		// redis.Nil is the special "key doesn't exist" error, which
		// here means expired or logged out. errors.Is asks "is this
		// error that specific one?" — safer than == because errors
		// are often wrapped inside others.
		if errors.Is(err, redis.Nil) {
			return nil, "", errNoSession
		}
		return nil, "", err
	}

	var s Session
	// Unmarshal parses the JSON into the struct. The & is required
	// because it WRITES INTO s, so it needs s's memory address.
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		return nil, "", err
	}

	// The absolute timeout — checked in code, because Redis only
	// knows about the sliding one.
	// Sub gives the gap between two times.
	if time.Since(s.CreatedAt) > absoluteTimeout {
		destroySession(ctx, token, s.UserID)
		return nil, "", errNoSession
	}

	// Sliding expiry: every valid request pushes the deadline out.
	// This is what makes it an IDLE timeout rather than a fixed one.
	rdb.Expire(ctx, sessionKey(h), idleTimeout)

	// & returns the address of s, so callers get a pointer.
	return &s, h, nil
}

func destroySession(ctx context.Context, token, userID string) error {
	h := hashToken(token)

	pipe := rdb.TxPipeline()
	pipe.Del(ctx, sessionKey(h))       // remove the session
	pipe.SRem(ctx, userKey(userID), h) // remove it from the user's set
	_, err := pipe.Exec(ctx)
	return err
}

// destroyAllForUser is "log out everywhere".
func destroyAllForUser(ctx context.Context, userID string) error {
	// SMembers reads every hash in the user's set.
	hashes, err := rdb.SMembers(ctx, userKey(userID)).Result()
	if err != nil {
		return err
	}

	pipe := rdb.TxPipeline()
	// range gives the index and value; _ discards the index.
	for _, h := range hashes {
		pipe.Del(ctx, sessionKey(h))
	}
	pipe.Del(ctx, userKey(userID))
	_, err = pipe.Exec(ctx)
	return err
}

// ---------------- COOKIES ----------------

func cookieName() string {
	if devMode {
		return cookieNameDev
	}
	return cookieNameSecure
}

// setCookie writes the session cookie with the security flags that
// actually matter.
func setCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:  cookieName(),
		Value: token,
		Path:  "/",

		// HttpOnly blocks JavaScript from reading it, so an XSS bug
		// can't steal the session. This single flag prevents the
		// most common real-world session theft.
		HttpOnly: true,

		// Secure means HTTPS only. Off in dev because localhost
		// isn't HTTPS.
		Secure: !devMode,

		// SameSite stops other sites triggering authenticated
		// requests on your behalf — the CSRF defence.
		SameSite: http.SameSiteLaxMode,

		// The browser drops it after this. Matches the absolute
		// timeout so the two clocks agree.
		MaxAge: int(absoluteTimeout.Seconds()),
	})
}

func clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName(),
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   !devMode,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // negative tells the browser to delete it now
	})
}

// ---------------- MIDDLEWARE ----------------

// A context key type of its own, so our value can't collide with
// another package's. struct{} is an empty type that costs no memory.
type ctxKey struct{}

// requireAuth wraps a handler, rejecting anyone without a session.
// It takes a function and returns a function — that's middleware.
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName())
		if err != nil {
			// No cookie at all.
			http.Error(w, "unauthorized", 401)
			return
		}

		s, _, err := getSession(r.Context(), c.Value)
		if err != nil {
			// Expired or revoked — clear the stale cookie so the
			// browser stops sending it.
			clearCookie(w)
			http.Error(w, "unauthorized", 401)
			return
		}

		// Attach the session to the request so the wrapped handler
		// can read it without looking it up again.
		// WithValue returns a NEW context; WithContext returns a
		// NEW request. Neither modifies the original.
		ctx := context.WithValue(r.Context(), ctxKey{}, s)

		// Call the real handler with the enriched request.
		next(w, r.WithContext(ctx))
	}
}

// Pull the session back out of the context.
func sessionFrom(r *http.Request) *Session {
	// The .(*Session) is a "type assertion" — the value is stored as
	// a general type, so we state what it really is.
	// The two-value form avoids a crash if it's missing.
	s, ok := r.Context().Value(ctxKey{}).(*Session)
	if !ok {
		return nil
	}
	return s
}

// ---------------- HANDLERS ----------------

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json", 400)
		return
	}

	// STAND-IN for real authentication. In production you'd look the
	// user up and compare a bcrypt/argon2 hash — never a plain
	// password, and never a plain == on secrets.
	// ConstantTimeCompare always takes the same time regardless of
	// where the values differ, so an attacker can't measure their
	// way to the answer.
	// It returns 1 for a match; the ==1 turns that into true/false.
	ok := subtle.ConstantTimeCompare([]byte(body.Password), []byte("hunter2")) == 1
	if !ok || body.Username == "" {
		// Deliberately vague: saying "no such user" would let
		// someone enumerate valid usernames.
		http.Error(w, "invalid credentials", 401)
		return
	}

	// IMPORTANT: if a session cookie already exists, destroy it
	// before making a new one. Reusing an ID across a login is
	// called session fixation — an attacker plants a known ID, you
	// log in, and they now hold an authenticated session.
	if c, err := r.Cookie(cookieName()); err == nil {
		if old, _, err := getSession(r.Context(), c.Value); err == nil {
			destroySession(r.Context(), c.Value, old.UserID)
		}
	}

	s := Session{
		UserID:    body.Username,
		CreatedAt: time.Now(),
		UserAgent: r.UserAgent(),
		IP:        clientIP(r),
	}

	token, err := createSession(r.Context(), s)
	if err != nil {
		http.Error(w, "could not create session", 500)
		return
	}

	setCookie(w, token)
	writeJSON(w, 200, map[string]string{"status": "logged in"})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(cookieName())
	if err == nil {
		if s, _, err := getSession(r.Context(), c.Value); err == nil {
			destroySession(r.Context(), c.Value, s.UserID)
		}
	}

	// Clear the cookie regardless. Logout must never fail — a user
	// who thinks they logged out but didn't is a security problem.
	clearCookie(w)
	writeJSON(w, 200, map[string]string{"status": "logged out"})
}

func handleLogoutAll(w http.ResponseWriter, r *http.Request) {
	s := sessionFrom(r)
	if err := destroyAllForUser(r.Context(), s.UserID); err != nil {
		http.Error(w, "could not log out", 500)
		return
	}
	clearCookie(w)
	writeJSON(w, 200, map[string]string{"status": "all sessions ended"})
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	s := sessionFrom(r)
	writeJSON(w, 200, map[string]any{
		"user_id":    s.UserID,
		"created_at": s.CreatedAt,
		"ip":         s.IP,
	})
}

func handleListSessions(w http.ResponseWriter, r *http.Request) {
	s := sessionFrom(r)

	hashes, err := rdb.SMembers(r.Context(), userKey(s.UserID)).Result()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Start empty (not nil) so an empty result becomes [] in JSON,
	// not null.
	out := []Session{}
	for _, h := range hashes {
		data, err := rdb.Get(r.Context(), sessionKey(h)).Result()
		if err != nil {
			continue // already expired — skip it
		}
		var sess Session
		if json.Unmarshal([]byte(data), &sess) == nil {
			out = append(out, sess)
		}
	}

	writeJSON(w, 200, out)
}

// ---------------- SMALL HELPERS ----------------

// clientIP finds the real address, allowing for a proxy in front.
func clientIP(r *http.Request) string {
	// ONLY trust this header if a proxy you control sets it —
	// clients can forge it freely.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// It's a comma-separated chain; the first entry is the client.
		// SplitN(s, ",", 2) cuts it into at most 2 pieces.
		return strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
	}
	// RemoteAddr looks like "1.2.3.4:5678" — cut off the port.
	// LastIndex finds the last colon, which handles IPv6 correctly.
	if i := strings.LastIndex(r.RemoteAddr, ":"); i > 0 {
		return r.RemoteAddr[:i] // [:i] means "up to position i"
	}
	return r.RemoteAddr
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	// Order matters: headers, then status, then body. Reversing it
	// makes Go send 200 and log "superfluous WriteHeader call".
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// Unused, kept to show the pattern for building keys safely.
var _ = fmt.Sprintf
