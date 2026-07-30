// Package config reads service configuration from the environment.
//
// Environment variables rather than files because every deployment target in
// this project injects config that way: docker-compose `environment:`, ECS task
// definitions, Kubernetes ConfigMaps and Lambda environment blocks all set env
// vars, and none of them mount a config file without extra machinery.
//
// The rule the API enforces: anything with a safe default gets one, and
// anything without a safe default must be Required. A service that silently
// starts with an empty database URL and fails on the first request is far
// worse than one that refuses to boot.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Loader accumulates errors while reading values so a misconfigured service
// reports every problem at once, rather than making the operator restart it
// repeatedly to discover one missing variable at a time.
type Loader struct {
	prefix string
	errs   []error
}

// New returns a Loader. A non-empty prefix is applied to every lookup, so
// New("CATALOG") resolves String("DB_URL", …) against CATALOG_DB_URL.
func New(prefix string) *Loader {
	return &Loader{prefix: prefix}
}

// key applies the prefix and normalises to the SCREAMING_SNAKE convention.
func (l *Loader) key(name string) string {
	name = strings.ToUpper(name)
	if l.prefix == "" {
		return name
	}
	return strings.ToUpper(l.prefix) + "_" + name
}

// Err returns all accumulated errors joined into one, or nil when the load was
// clean. Call it once after reading every field.
func (l *Loader) Err() error {
	if len(l.errs) == 0 {
		return nil
	}
	msgs := make([]string, len(l.errs))
	for i, err := range l.errs {
		msgs[i] = err.Error()
	}
	return fmt.Errorf("config: %d problem(s):\n  - %s", len(l.errs), strings.Join(msgs, "\n  - "))
}

// String returns the variable's value, or def when unset or empty.
func (l *Loader) String(name, def string) string {
	if v, ok := lookup(l.key(name)); ok {
		return v
	}
	return def
}

// Required returns the variable's value and records an error when it is unset.
// Use it for anything with no safe default: database URLs, broker addresses,
// bucket names.
func (l *Loader) Required(name string) string {
	v, ok := lookup(l.key(name))
	if !ok {
		l.errs = append(l.errs, fmt.Errorf("%s is required but not set", l.key(name)))
		return ""
	}
	return v
}

// Int returns the variable parsed as an int, or def when unset. A value that
// is present but unparseable is an error, never a silent fallback to def --
// PORT=eighty is a typo the operator needs told about.
func (l *Loader) Int(name string, def int) int {
	raw, ok := lookup(l.key(name))
	if !ok {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s = %q is not a valid integer", l.key(name), raw))
		return def
	}
	return v
}

// Bool accepts the forms strconv.ParseBool does: 1, t, T, true, TRUE, 0, f,
// false and so on.
func (l *Loader) Bool(name string, def bool) bool {
	raw, ok := lookup(l.key(name))
	if !ok {
		return def
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s = %q is not a valid boolean", l.key(name), raw))
		return def
	}
	return v
}

// Duration parses Go duration syntax: "30s", "5m", "1h30m".
func (l *Loader) Duration(name string, def time.Duration) time.Duration {
	raw, ok := lookup(l.key(name))
	if !ok {
		return def
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s = %q is not a valid duration (want e.g. \"30s\")", l.key(name), raw))
		return def
	}
	return v
}

// lookup treats an empty value as unset. An env var explicitly set to "" is
// almost always an unsubstituted template variable rather than a deliberate
// empty string, so falling back to the default is the safer reading.
func lookup(key string) (string, bool) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}
