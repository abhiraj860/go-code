package config

import (
	"strings"
	"testing"
	"time"
)

func TestStringAndDefaults(t *testing.T) {
	t.Setenv("APP_NAME", "catalog-svc")

	l := New("")
	if got := l.String("APP_NAME", "fallback"); got != "catalog-svc" {
		t.Errorf("String = %q, want catalog-svc", got)
	}
	if got := l.String("APP_MISSING", "fallback"); got != "fallback" {
		t.Errorf("String on unset var = %q, want fallback", got)
	}
	if err := l.Err(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrefixIsApplied(t *testing.T) {
	t.Setenv("CATALOG_DB_URL", "postgres://localhost/catalog")

	l := New("catalog") // lower-case prefix must still resolve
	if got := l.String("db_url", ""); got != "postgres://localhost/catalog" {
		t.Errorf("prefixed lookup = %q, want the catalog URL", got)
	}
}

func TestRequiredRecordsErrorWhenUnset(t *testing.T) {
	l := New("")
	l.Required("DEFINITELY_NOT_SET_XYZ")

	err := l.Err()
	if err == nil {
		t.Fatal("Required on an unset var did not produce an error")
	}
	if !strings.Contains(err.Error(), "DEFINITELY_NOT_SET_XYZ") {
		t.Errorf("error does not name the variable: %v", err)
	}
}

func TestEmptyValueTreatedAsUnset(t *testing.T) {
	// An env var set to "" is nearly always an unsubstituted template
	// variable, so the default should win rather than an empty string
	// propagating into a connection string.
	t.Setenv("EMPTY_VAR", "")

	l := New("")
	if got := l.String("EMPTY_VAR", "fallback"); got != "fallback" {
		t.Errorf("String on empty var = %q, want fallback", got)
	}

	l2 := New("")
	l2.Required("EMPTY_VAR")
	if l2.Err() == nil {
		t.Error("Required treated an empty value as present")
	}
}

func TestIntParsing(t *testing.T) {
	t.Setenv("PORT", "8080")
	t.Setenv("BAD_PORT", "eighty")

	l := New("")
	if got := l.Int("PORT", 1); got != 8080 {
		t.Errorf("Int = %d, want 8080", got)
	}
	if got := l.Int("UNSET_PORT", 9090); got != 9090 {
		t.Errorf("Int on unset var = %d, want 9090", got)
	}
	if err := l.Err(); err != nil {
		t.Fatalf("unexpected error before the bad parse: %v", err)
	}

	// Present-but-unparseable must be an error, not a silent default.
	l.Int("BAD_PORT", 1)
	if l.Err() == nil {
		t.Error("Int did not report an unparseable value")
	}
}

func TestDurationParsing(t *testing.T) {
	t.Setenv("HOLD_TTL", "2m30s")

	l := New("")
	if got, want := l.Duration("HOLD_TTL", time.Second), 150*time.Second; got != want {
		t.Errorf("Duration = %v, want %v", got, want)
	}
	if got := l.Duration("UNSET_TTL", 5*time.Second); got != 5*time.Second {
		t.Errorf("Duration on unset var = %v, want 5s", got)
	}
	if err := l.Err(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBoolParsing(t *testing.T) {
	tests := []struct {
		raw  string
		want bool
	}{
		{"true", true}, {"TRUE", true}, {"1", true}, {"t", true},
		{"false", false}, {"0", false}, {"f", false},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			t.Setenv("FLAG", tt.raw)
			l := New("")
			if got := l.Bool("FLAG", !tt.want); got != tt.want {
				t.Errorf("Bool(%q) = %v, want %v", tt.raw, got, tt.want)
			}
			if err := l.Err(); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// The point of accumulating rather than failing fast: one restart should
// surface every misconfiguration, not just the first.
func TestErrorsAccumulate(t *testing.T) {
	t.Setenv("BAD_INT", "abc")
	t.Setenv("BAD_DUR", "5 minutes")

	l := New("")
	l.Required("MISSING_ONE")
	l.Required("MISSING_TWO")
	l.Int("BAD_INT", 0)
	l.Duration("BAD_DUR", 0)

	err := l.Err()
	if err == nil {
		t.Fatal("expected accumulated errors")
	}
	msg := err.Error()
	for _, want := range []string{"MISSING_ONE", "MISSING_TWO", "BAD_INT", "BAD_DUR"} {
		if !strings.Contains(msg, want) {
			t.Errorf("accumulated error is missing %q:\n%s", want, msg)
		}
	}
	if !strings.Contains(msg, "4 problem(s)") {
		t.Errorf("expected a count of 4 problems, got:\n%s", msg)
	}
}
