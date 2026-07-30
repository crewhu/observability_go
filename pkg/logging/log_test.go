package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	otellog "go.opentelemetry.io/otel/log"
	otellognoop "go.opentelemetry.io/otel/log/noop"
)

// syncBuffer lets concurrent test goroutines share one destination safely;
// it does not exercise slog's own internal write locking.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := strings.TrimSpace(b.buf.String())
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func TestLogCallSiteTagsAppearInJSONOutput(t *testing.T) {
	prevLogger := logger
	defer func() { logger = prevLogger }()

	buf := &syncBuffer{}
	ConfigureLoggerWithWriter(buf, LogLevelInfo)

	ctx := WithTag(context.Background(), "handler", "TestHandler")
	Info(ctx, "finished", Tags{"contacts_upserted": 42})

	lines := buf.Lines()
	if len(lines) != 1 {
		t.Fatalf("want 1 log line, got %d: %v", len(lines), lines)
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("log line is not valid JSON: %v (%s)", err, lines[0])
	}

	if entry["msg"] != "finished" {
		t.Fatalf("got msg=%v, want %q", entry["msg"], "finished")
	}
	if entry["handler"] != "TestHandler" {
		t.Fatalf("missing ctx tag %q in output: %v", "handler", entry)
	}
	if entry["contacts_upserted"] != float64(42) {
		t.Fatalf("missing call-site tag %q in output: %v", "contacts_upserted", entry)
	}
}

// TestLogTagsDoNotCrossContaminateConcurrentHandlers is the regression test
// for the Signoz symptom: two sibling handler goroutines tagging from the
// same parent ctx must each produce a log line carrying only their own tags.
func TestLogTagsDoNotCrossContaminateConcurrentHandlers(t *testing.T) {
	prevLogger := logger
	defer func() { logger = prevLogger }()

	buf := &syncBuffer{}
	ConfigureLoggerWithWriter(buf, LogLevelInfo)

	parent := context.Background()
	handlers := []string{"TicketsHandler", "ProjectsHandler"}

	var wg sync.WaitGroup
	for _, name := range handlers {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			ctx := WithTag(parent, "handler", name)
			Info(ctx, "finished", Tags{"upserted": len(name)})
		}(name)
	}
	wg.Wait()

	lines := buf.Lines()
	if len(lines) != len(handlers) {
		t.Fatalf("want %d log lines, got %d: %v", len(handlers), len(lines), lines)
	}

	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("log line is not valid JSON: %v (%s)", err, line)
		}
		handler, _ := entry["handler"].(string)
		wantUpserted := float64(len(handler))
		if entry["upserted"] != wantUpserted {
			t.Fatalf("handler %q log line carries a mismatched upserted value (cross-contamination): %v", handler, entry)
		}
	}
}

type recordingOtelLogger struct {
	otellognoop.Logger
	mu      sync.Mutex
	records []otellog.Record
}

func (r *recordingOtelLogger) Emit(_ context.Context, record otellog.Record) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, record)
}

func TestLogCallSiteTagsAppearInOtelRecord(t *testing.T) {
	prev := otelLogger
	rec := &recordingOtelLogger{}
	otelLogger = rec
	defer func() { otelLogger = prev }()

	ctx := WithTag(context.Background(), "handler", "TestHandler")
	Info(ctx, "finished", Tags{"contacts_upserted": 42})

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.records) != 1 {
		t.Fatalf("want 1 otel record, got %d", len(rec.records))
	}

	got := map[string]string{}
	rec.records[0].WalkAttributes(func(kv otellog.KeyValue) bool {
		got[kv.Key] = kv.Value.AsString()
		return true
	})

	if got["handler"] != "TestHandler" {
		t.Fatalf("otel record missing ctx tag %q: %+v", "handler", got)
	}
	if got["contacts_upserted"] != "42" {
		t.Fatalf("otel record missing call-site tag %q: %+v", "contacts_upserted", got)
	}
}

// TestLogCallSiteTagOverridesCollidingCtxTag guards against a regression
// where a call-site Tag sharing a key with a ctx tag (e.g. "handler") got
// applied via a second, independent With()/AddAttributes() call and ended up
// duplicated in the output instead of overriding the ctx value. json.Unmarshal
// into a map silently collapses duplicate keys, so this asserts on the raw
// JSON bytes as well as the parsed value.
func TestLogCallSiteTagOverridesCollidingCtxTag(t *testing.T) {
	prevLogger := logger
	defer func() { logger = prevLogger }()
	prevOtel := otelLogger
	rec := &recordingOtelLogger{}
	otelLogger = rec
	defer func() { otelLogger = prevOtel }()

	buf := &syncBuffer{}
	ConfigureLoggerWithWriter(buf, LogLevelInfo)

	ctx := WithTag(context.Background(), "handler", "TicketsHandler")
	Info(ctx, "processing", Tags{"handler": "override-value"})

	lines := buf.Lines()
	if len(lines) != 1 {
		t.Fatalf("want 1 log line, got %d: %v", len(lines), lines)
	}
	if n := strings.Count(lines[0], `"handler"`); n != 1 {
		t.Fatalf("want exactly 1 %q key in JSON output, got %d: %s", "handler", n, lines[0])
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("log line is not valid JSON: %v (%s)", err, lines[0])
	}
	if entry["handler"] != "override-value" {
		t.Fatalf("call-site tag should override the ctx tag, got handler=%v", entry["handler"])
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.records) != 1 {
		t.Fatalf("want 1 otel record, got %d", len(rec.records))
	}
	count := 0
	var otelValue string
	rec.records[0].WalkAttributes(func(kv otellog.KeyValue) bool {
		if kv.Key == "handler" {
			count++
			otelValue = kv.Value.AsString()
		}
		return true
	})
	if count != 1 {
		t.Fatalf("want exactly 1 otel %q attribute, got %d", "handler", count)
	}
	if otelValue != "override-value" {
		t.Fatalf("otel record should carry the call-site override, got %q", otelValue)
	}
}

// TestErrDoesNotDuplicateCollidingCtxErrorTag targets Err()'s hardcoded
// Tags{"error": true}: if a caller already tagged the ctx with "error"
// upstream (a guessable key), Err() must not double-emit the attribute.
func TestErrDoesNotDuplicateCollidingCtxErrorTag(t *testing.T) {
	prevLogger := logger
	defer func() { logger = prevLogger }()

	buf := &syncBuffer{}
	ConfigureLoggerWithWriter(buf, LogLevelInfo)

	ctx := WithTag(context.Background(), "error", "upstream-flag")
	Err(ctx, errors.New("boom"))

	lines := buf.Lines()
	if len(lines) != 1 {
		t.Fatalf("want 1 log line, got %d: %v", len(lines), lines)
	}
	if n := strings.Count(lines[0], `"error"`); n != 1 {
		t.Fatalf("want exactly 1 %q key in JSON output, got %d: %s", "error", n, lines[0])
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("log line is not valid JSON: %v (%s)", err, lines[0])
	}
	if entry["error"] != true {
		t.Fatalf("Err's hardcoded error=true tag should win, got error=%v", entry["error"])
	}
}
