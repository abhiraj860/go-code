// Package generator turns an order into printable tickets.
//
// This is the CPU-bound half of ticket-svc, and the reason the consumer runs a
// worker pool: rendering a PDF is milliseconds of pure computation with no I/O
// to overlap, so parallelism here is what keeps a drop's worth of orders from
// queueing behind each other.
package generator

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/skip2/go-qrcode"
)

// Ticket is one seat's admission document.
type Ticket struct {
	ID      string
	OrderID string
	EventID string
	SeatID  string
	UserID  string
	// Code is what the venue scanner reads. Signed, so a forged QR fails
	// validation at the gate without a database lookup.
	Code string
}

// Generator renders tickets.
type Generator struct {
	// signingKey authenticates gate codes. A ticket QR is a bearer token: with
	// an unsigned code, anyone who can guess an id gets into the venue.
	signingKey []byte
	eventTitle func(eventID string) string
}

type Options struct {
	// SigningKey must be at least 32 bytes. Required -- there is no safe
	// default for a key, and generating one per process would invalidate every
	// ticket on restart.
	SigningKey []byte
	// EventTitle resolves a display title. Optional; falls back to the id.
	EventTitle func(eventID string) string
}

func New(opts Options) (*Generator, error) {
	if len(opts.SigningKey) < 32 {
		return nil, errors.New("generator: signing key must be at least 32 bytes")
	}
	if opts.EventTitle == nil {
		opts.EventTitle = func(id string) string { return id }
	}
	return &Generator{signingKey: opts.SigningKey, eventTitle: opts.EventTitle}, nil
}

// GateCode returns the signed payload encoded in a ticket's QR.
//
// Format: base64(ticketID|eventID|seatID).base64(hmac). The gate verifies the
// signature locally, so a scanner keeps working through a network partition --
// which matters when 20,000 people are queueing outside a stadium.
func (g *Generator) GateCode(ticketID, eventID, seatID string) string {
	payload := fmt.Sprintf("%s|%s|%s", ticketID, eventID, seatID)

	mac := hmac.New(sha256.New, g.signingKey)
	mac.Write([]byte(payload))

	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." +
		base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// VerifyGateCode checks a scanned code and returns its payload fields.
func (g *Generator) VerifyGateCode(code string) (ticketID, eventID, seatID string, err error) {
	rawPayload, rawMAC, ok := cut(code, '.')
	if !ok {
		return "", "", "", errors.New("generator: malformed gate code")
	}

	payload, err := base64.RawURLEncoding.DecodeString(rawPayload)
	if err != nil {
		return "", "", "", errors.New("generator: malformed gate code payload")
	}
	sig, err := base64.RawURLEncoding.DecodeString(rawMAC)
	if err != nil {
		return "", "", "", errors.New("generator: malformed gate code signature")
	}

	mac := hmac.New(sha256.New, g.signingKey)
	mac.Write(payload)
	// Constant-time compare: a byte-by-byte comparison leaks how much of a
	// forged signature was correct, which is enough to forge one incrementally.
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return "", "", "", errors.New("generator: gate code signature is invalid")
	}

	parts := splitN(string(payload), '|', 3)
	if len(parts) != 3 {
		return "", "", "", errors.New("generator: gate code payload is malformed")
	}
	return parts[0], parts[1], parts[2], nil
}

// QR renders the gate code as a PNG.
func (g *Generator) QR(code string) ([]byte, error) {
	// Medium recovery: enough to survive a crease or a thumb on a phone screen,
	// without the density that makes a cheap scanner struggle.
	png, err := qrcode.Encode(code, qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("generator: encoding qr: %w", err)
	}
	return png, nil
}

// PDF renders a printable ticket.
func (g *Generator) PDF(t Ticket, startsAt time.Time, venue string) ([]byte, error) {
	qr, err := g.QR(t.Code)
	if err != nil {
		return nil, err
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 22)
	pdf.CellFormat(0, 14, "TicketFlow", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 12, g.eventTitle(t.EventID), "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 11)
	if venue != "" {
		pdf.CellFormat(0, 7, venue, "", 1, "L", false, 0, "")
	}
	if !startsAt.IsZero() {
		pdf.CellFormat(0, 7, startsAt.Format("Mon 2 Jan 2006, 15:04 MST"), "", 1, "L", false, 0, "")
	}

	pdf.Ln(4)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 9, "Seat "+t.SeatID, "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 6, "Ticket "+t.ID, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, "Order "+t.OrderID, "", 1, "L", false, 0, "")

	// Register the PNG from memory rather than a temp file, so generation stays
	// pure computation and several workers cannot collide on a shared path.
	pdf.RegisterImageOptionsReader("qr", fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(qr))
	pdf.ImageOptions("qr", 20, pdf.GetY()+6, 55, 55, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")

	pdf.SetY(pdf.GetY() + 68)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.CellFormat(0, 5, "Present this code at the gate. Admits one.", "", 1, "L", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generator: writing pdf: %w", err)
	}
	return buf.Bytes(), nil
}

// PDFKey is the blob key for a ticket's PDF.
//
// Derived deterministically from the ticket id so a redelivered order.created
// overwrites the same object instead of littering the bucket with duplicates --
// which also makes an Exists check a valid idempotency guard.
func PDFKey(eventID, ticketID string) string {
	return fmt.Sprintf("tickets/%s/%s.pdf", eventID, ticketID)
}

// TicketID is deterministic in the order and seat, so regenerating from a
// redelivered message produces the SAME ticket id rather than a second ticket
// for one seat.
func TicketID(orderID, seatID string) string {
	sum := sha256.Sum256([]byte(orderID + "|" + seatID))
	return "tkt_" + base64.RawURLEncoding.EncodeToString(sum[:12])
}

func cut(s string, sep byte) (before, after string, found bool) {
	for i := range len(s) {
		if s[i] == sep {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}

func splitN(s string, sep byte, n int) []string {
	out := make([]string, 0, n)
	start := 0
	for i := range len(s) {
		if s[i] == sep && len(out) < n-1 {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}
