package main

import (
	"context"       // carries cancellation signals between function calls
	"errors"        // create and inspect error values
	"fmt"           // build formatted strings
	"log"           // print messages
	"os"            // read environment variables
	"os/signal"     // catch Ctrl+C so we can shut down cleanly
	"syscall"       // the specific signal names to catch
	"time"          // durations and delays

	"github.com/jackc/pglogrepl"          // speaks Postgres's replication protocol
	"github.com/jackc/pgx/v5/pgconn"      // low-level connection, needed for replication
	"github.com/jackc/pgx/v5/pgproto3"    // the raw message types Postgres sends
	"github.com/jackc/pgx/v5/pgxpool"     // normal connection pool for regular queries
)

// Names we use throughout. Constants can't be changed at runtime.
const (
	slotName    = "cdc_prod_slot"  // our permanent bookmark in Postgres
	publication = "cdc_prod_pub"   // which tables we want changes from
	// How often we tell Postgres "still alive, still reading".
	standbyInterval = 10 * time.Second
)

// One decoded row change. This is what our business logic consumes.
type Change struct {
	Op    string            // "INSERT", "UPDATE" or "DELETE"
	Table string            // e.g. "public.todos"
	New   map[string]string // column name -> value after the change
	Old   map[string]string // column name -> value before it (UPDATE/DELETE)
}

func main() {
	// NotifyContext gives a context that cancels when Ctrl+C is pressed.
	// This is how we shut down without losing in-flight work.
	// The second return value is a cleanup function we defer.
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop() // defer = run this when main() ends, whatever happens

	dsn := os.Getenv("DATABASE_URL")
	queryDSN := os.Getenv("DATABASE_URL_QUERY")
	if dsn == "" || queryDSN == "" {
		log.Fatal("set DATABASE_URL and DATABASE_URL_QUERY")
	}


	// A NORMAL connection pool, separate from the replication one.
	// We need this because a replication connection can't run queries,
	// and we need queries to store our progress.
	pool, err := pgxpool.New(ctx, queryDSN)
	if err != nil {
		log.Fatal("pool: ", err)
	}
	defer pool.Close()

	// Create the table where we remember how far we've processed.
	if err := ensureOffsetTable(ctx, pool); err != nil {
		log.Fatal("offset table: ", err)
	}

	// Create the publication — Postgres's list of tables to watch.
	if err := ensurePublication(ctx, pool); err != nil {
		log.Fatal("publication: ", err)
	}

	// Reconnect loop. A network blip should not kill the process,
	// so we retry forever with a growing delay between attempts.
	backoff := time.Second

	for {
		// ctx.Err() is non-nil once Ctrl+C was pressed.
		if ctx.Err() != nil {
			log.Println("shutting down")
			return
		}

		err := stream(ctx, dsn, pool)

		// Context cancelled means we asked to stop — not a failure.
		if errors.Is(err, context.Canceled) {
			log.Println("stopped cleanly")
			return
		}

		log.Println("stream ended:", err)

		log.Println("reconnecting in", backoff)

		// A select waits on several things at once and takes whichever
		// happens first. Here: either the delay finishes, or we're
		// asked to shut down — so Ctrl+C doesn't wait out the delay.
		select {
		case <-time.After(backoff): // a channel that fires after the delay
		case <-ctx.Done():          // fires immediately on shutdown
			return
		}

		// Double the wait each time, capped, so we don't hammer a
		// database that's genuinely down. This is called backoff.
		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}
}

// ---------------- SETUP ----------------

func ensureOffsetTable(ctx context.Context, pool *pgxpool.Pool) error {
	// IF NOT EXISTS makes this safe to run on every startup.
	// The backticks make a raw string, so we can write multi-line SQL
	// without escaping anything.
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS cdc_offsets (
			slot_name TEXT PRIMARY KEY,
			lsn       TEXT NOT NULL,
			updated   TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	// The _ discards the first return value (a result summary)
	// because we only care whether it errored.
	return err
}

func ensurePublication(ctx context.Context, pool *pgxpool.Pool) error {
	var exists bool

	// $1 is a placeholder Postgres fills with the value below —
	// this is what makes the query safe from injection.
	// Scan writes the answer into exists, so it needs its ADDRESS,
	// which is what & gives us.
	err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_publication WHERE pubname = $1)`,
		publication).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		return nil // nil means "no error"
	}

	// FOR ALL TABLES watches everything. In production you'd usually
	// name specific tables instead: FOR TABLE public.todos, public.orders
	// Sprintf builds the string because table/publication names can't
	// be passed as $1 placeholders — only values can.
	_, err = pool.Exec(ctx,
		fmt.Sprintf("CREATE PUBLICATION %s FOR ALL TABLES", publication))
	return err
}

// ---------------- OFFSET STORAGE ----------------

// Read where we left off last time we ran.
func loadOffset(ctx context.Context, pool *pgxpool.Pool) (pglogrepl.LSN, error) {
	var s string

	err := pool.QueryRow(ctx,
		`SELECT lsn FROM cdc_offsets WHERE slot_name = $1`, slotName).Scan(&s)

	// No row means this is our first ever run. Returning 0 tells
	// Postgres "start wherever the slot says", which is correct.
	if err != nil {
		return 0, nil
	}

	// An LSN is a position in the log, stored as text like "0/1A2B3C0".
	// ParseLSN turns that text back into the number type.
	return pglogrepl.ParseLSN(s)
}

// ---------------- THE STREAM ----------------

func stream(ctx context.Context, dsn string, pool *pgxpool.Pool) error {
	// A raw replication connection. The DSN must contain
	// replication=database or this becomes a normal connection.
	conn, err := pgconn.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", err) // %w wraps the original error
	}
	defer conn.Close(ctx)

	// Create the slot if it isn't there. NOT temporary this time —
	// a permanent slot survives restarts, which is the whole point.
	err = createSlotIfMissing(ctx, conn)
	if err != nil {
		return err
	}

	// Where to resume from.
	startPos, err := loadOffset(ctx, pool)
	if err != nil {
		return fmt.Errorf("load offset: %w", err)
	}
	log.Println("resuming from", startPos)

	// pgoutput is Postgres's built-in binary format — the production
	// choice. PluginArgs tell it which protocol version and which
	// publication to send.
	err = pglogrepl.StartReplication(ctx, conn, slotName, startPos,
		pglogrepl.StartReplicationOptions{
			PluginArgs: []string{
				"proto_version '1'",
				fmt.Sprintf("publication_names '%s'", publication),
			},
		})
	if err != nil {
		return fmt.Errorf("start replication: %w", err)
	}
	log.Println("streaming")

	// The position we've CONFIRMED processing. This is what we report
	// back, and it only moves forward after work actually succeeds.
	confirmed := startPos

	// Relations describe table structure (column names and types).
	// Postgres sends them once, then refers to tables by a number,
	// so we cache them. map[uint32]* means "number -> pointer to".
	relations := map[uint32]*pglogrepl.RelationMessage{}

	// Changes collected for the transaction currently being read.
	// We hold them until COMMIT so a half-written transaction is
	// never handed to our business logic.
	var batch []Change

	nextDeadline := time.Now().Add(standbyInterval)

	for {
		// Send the heartbeat when due. This tells Postgres it can
		// discard WAL up to `confirmed`. Skip it and the disk fills.
		if time.Now().After(nextDeadline) {
			err = pglogrepl.SendStandbyStatusUpdate(ctx, conn,
				pglogrepl.StandbyStatusUpdate{WALWritePosition: confirmed})
			if err != nil {
				return fmt.Errorf("standby update: %w", err)
			}
			nextDeadline = time.Now().Add(standbyInterval)
		}

		// A context that expires when the heartbeat is next due, so
		// a quiet database doesn't block us past our deadline.
		recvCtx, cancel := context.WithDeadline(ctx, nextDeadline)
		rawMsg, err := conn.ReceiveMessage(recvCtx)
		cancel() // release it immediately, don't wait for the loop to end

		if err != nil {
			// Our own deadline expiring is normal — loop round.
			if pgconn.Timeout(err) {
				continue
			}
			// Shutdown requested — pass it up so main() knows.
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("receive: %w", err)
		}

		// A type switch branches on what type a value actually is.
		switch msg := rawMsg.(type) {

		case *pgproto3.ErrorResponse:
			return fmt.Errorf("server error: %s", msg.Message)

		case *pgproto3.CopyData:
			// The first byte says what kind of payload follows.
			// [0] is that byte; [1:] is everything after it.
			switch msg.Data[0] {

			case pglogrepl.PrimaryKeepaliveMessageByteID:
				pkm, err := pglogrepl.ParsePrimaryKeepaliveMessage(msg.Data[1:])
				if err != nil {
					return err
				}
				// If Postgres explicitly wants an answer, make the
				// heartbeat due right now.
				if pkm.ReplyRequested {
					nextDeadline = time.Time{}
				}

			case pglogrepl.XLogDataByteID:
				xld, err := pglogrepl.ParseXLogData(msg.Data[1:])
				if err != nil {
					return err
				}

				// Decode the binary pgoutput message.
				logicalMsg, err := pglogrepl.Parse(xld.WALData)
				if err != nil {
					return fmt.Errorf("parse: %w", err)
				}

				// Handle each message type.
				switch m := logicalMsg.(type) {

				// Table structure — remember it for later decoding.
				case *pglogrepl.RelationMessage:
					relations[m.RelationID] = m

				// Start of a transaction — clear the batch.
				// batch[:0] reuses the existing memory instead of
				// allocating a new slice.
				case *pglogrepl.BeginMessage:
					batch = batch[:0]

				case *pglogrepl.InsertMessage:
					rel := relations[m.RelationID]
					if rel == nil {
						continue // haven't seen this table's structure yet
					}
					batch = append(batch, Change{
						Op:    "INSERT",
						Table: rel.Namespace + "." + rel.RelationName,
						New:   decodeTuple(rel, m.Tuple),
					})

				case *pglogrepl.UpdateMessage:
					rel := relations[m.RelationID]
					if rel == nil {
						continue
					}
					batch = append(batch, Change{
						Op:    "UPDATE",
						Table: rel.Namespace + "." + rel.RelationName,
						// OldTuple is only populated when the table has
						// REPLICA IDENTITY FULL set.
						Old: decodeTuple(rel, m.OldTuple),
						New: decodeTuple(rel, m.NewTuple),
					})

				case *pglogrepl.DeleteMessage:
					rel := relations[m.RelationID]
					if rel == nil {
						continue
					}
					batch = append(batch, Change{
						Op:    "DELETE",
						Table: rel.Namespace + "." + rel.RelationName,
						Old:   decodeTuple(rel, m.OldTuple),
					})

				// End of transaction — NOW we do the work.
				case *pglogrepl.CommitMessage:
					// This is the critical ordering: process the whole
					// batch AND save the new offset in one database
					// transaction. Either both happen or neither does.
					if err := handleBatch(ctx, pool, batch, m.CommitLSN); err != nil {
						// Do NOT advance `confirmed`. We'll reconnect
						// and Postgres will resend from where we are.
						return fmt.Errorf("handle batch: %w", err)
					}

					// Only after success do we move our position.
					confirmed = m.CommitLSN
					batch = batch[:0]
				}
			}
		}
	}
}

// ---------------- SLOT CREATION ----------------

func createSlotIfMissing(ctx context.Context, conn *pgconn.PgConn) error {
	_, err := pglogrepl.CreateReplicationSlot(ctx, conn, slotName, "pgoutput",
		pglogrepl.CreateReplicationSlotOptions{Temporary: false})
	if err == nil {
		log.Println("created slot", slotName)
		return nil
	}

	// Already existing is expected on every run after the first.
	// errors.As checks whether the error is (or wraps) a PgError,
	// and if so writes it into pgErr — hence the &.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42710" { // duplicate_object
		log.Println("slot already exists — reusing")
		return nil
	}

	return fmt.Errorf("create slot: %w", err)
}

// ---------------- DECODING ----------------

// Turn Postgres's positional column data into a name -> value map.
func decodeTuple(rel *pglogrepl.RelationMessage, tuple *pglogrepl.TupleData) map[string]string {
	// nil means "not provided" — normal for UPDATE without
	// REPLICA IDENTITY FULL.
	if tuple == nil {
		return nil
	}

	out := map[string]string{}

	// range gives us the index (i) and the value (col).
	for i, col := range tuple.Columns {
		// Guard against a structure change mid-stream.
		if i >= len(rel.Columns) {
			break
		}
		name := rel.Columns[i].Name

		// DataType is a single character telling us what kind of
		// value this is. Comparing to 'n' uses single quotes because
		// it's one character, not a string.
		switch col.DataType {
		case 'n': // null
			out[name] = ""
		case 'u': // unchanged large value, not sent to save bandwidth
			out[name] = "<unchanged>"
		case 't': // actual text data
			// string(...) converts raw bytes into readable text.
			out[name] = string(col.Data)
		}
	}
	return out
}

// ---------------- YOUR BUSINESS LOGIC ----------------

// handleBatch is where real work goes. The important part is that the
// work and the offset save happen in ONE transaction.
func handleBatch(ctx context.Context, pool *pgxpool.Pool, batch []Change, commitLSN pglogrepl.LSN) error {
	// Begin starts a transaction: a group of statements that all
	// succeed together or all get undone together.
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	// Rollback undoes everything if we return early. Once Commit has
	// run, this call does nothing, so it's safe to always defer it.
	defer tx.Rollback(ctx)

	for _, c := range batch {
		// Replace this with your real work — writing to another
		// store, publishing to Kafka, invalidating a cache.
		log.Printf("%s %s new=%v old=%v", c.Op, c.Table, c.New, c.Old)
	}

	// Save our new position in the SAME transaction as the work above.
	// ON CONFLICT ... DO UPDATE means "insert, or update if the key
	// already exists" — often called an upsert.
	// EXCLUDED refers to the row we tried to insert.
	_, err = tx.Exec(ctx, `
		INSERT INTO cdc_offsets (slot_name, lsn, updated)
		VALUES ($1, $2, now())
		ON CONFLICT (slot_name)
		DO UPDATE SET lsn = EXCLUDED.lsn, updated = now()`,
		slotName, commitLSN.String())
	if err != nil {
		return err
	}

	// Commit makes everything permanent. If this fails, nothing above
	// took effect and we'll safely reprocess after reconnecting.
	return tx.Commit(ctx)
}