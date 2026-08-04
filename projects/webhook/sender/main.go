package main

import (
	"bytes"         // wrap bytes so they can be read like a file
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var secret = os.Getenv("WEBHOOK_SECRET")

// Where we're sending to.
const target = "http://localhost:9000/webhook"

func main() {
	if secret == "" {
		log.Fatal("set WEBHOOK_SECRET")
	}

	// map[string]any means "string keys, values of any type at all".
	// Using a map here (instead of a struct) keeps the example short.
	payload := map[string]any{
		"event": "order.updated",
		// UnixNano gives a number that's different every run, so each
		// send gets a unique ID and isn't treated as a duplicate.
		"id": fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		"data": map[string]any{
			"order_id": 1042,
			"status":   "shipped",
		},
	}

	// Marshal turns the map into JSON bytes.
	body, err := json.Marshal(payload)
	if err != nil {
		log.Fatal(err)
	}

	if err := sendWithRetry(body); err != nil {
		log.Fatal("giving up: ", err)
	}
}

func sendWithRetry(body []byte) error {
	// A client with a timeout. The default http.Client waits FOREVER,
	// which will eventually hang your program.
	client := &http.Client{Timeout: 5 * time.Second}

	// Try up to 4 times, waiting longer after each failure.
	for attempt := 1; attempt <= 4; attempt++ {
		// A fresh timestamp each attempt, so retries stay within
		// the receiver's freshness window.
		ts := fmt.Sprintf("%d", time.Now().Unix())

		// bytes.NewReader wraps our bytes so the request can read
		// from them, like a tiny in-memory file. We rebuild it every
		// attempt because a reader is consumed once it's been read.
		req, err := http.NewRequest("POST", target, bytes.NewReader(body))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Timestamp", ts)
		req.Header.Set("X-Signature", sign(body, ts))

		res, err := client.Do(req) // actually send it

		// err != nil means the network failed — worth retrying.
		if err != nil {
			log.Printf("attempt %d: %v", attempt, err)
		} else {
			// Always close the body or the connection leaks.
			res.Body.Close()

			// 2xx means success. Integer division by 100 turns
			// 200/201/204 all into 2.
			if res.StatusCode/100 == 2 {
				log.Println("delivered on attempt", attempt)
				return nil // nil means "no error"
			}

			// 4xx means WE sent something wrong — retrying the same
			// bad request will fail identically, so stop now.
			if res.StatusCode/100 == 4 {
				return fmt.Errorf("permanent failure: %d", res.StatusCode)
			}

			// 5xx means their server broke; it might recover.
			log.Printf("attempt %d: status %d", attempt, res.StatusCode)
		}

		// Wait longer after each failure so we don't hammer a
		// struggling server. 1s, 2s, 4s — this is called backoff.
		// 1<<attempt means 1 shifted left, giving 2, 4, 8...
		wait := time.Duration(1<<(attempt-1)) * time.Second
		log.Println("retrying in", wait)
		time.Sleep(wait)
	}

	return fmt.Errorf("all attempts failed")
}

// Identical to the receiver's verify logic — both sides must compute
// the signature exactly the same way or nothing will ever match.
func sign(body []byte, ts string) string {
	signed := fmt.Sprintf("%s.%s", ts, string(body))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed))
	return hex.EncodeToString(mac.Sum(nil))
}