// CRC: crc-SessionRef.md | Test: test-SessionRef.md
package server

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/zot/ui-engine/internal/config"
)

// captureLog redirects the standard logger for the test's duration.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}

func answer() (interface{}, error) { return 42, nil }

// Test: a dangerous call warns once per site (R168, R170)
func TestDangerousCallWarnsOncePerSite(t *testing.T) {
	s := newRefTestServer(t, nil)
	_, vendedID := newSession(t, s)
	buf := captureLog(t)

	for i := 0; i < 3; i++ {
		if got, err := s.ExecuteInSession(vendedID, answer); err != nil || got != 42 {
			t.Fatalf("call %d returned %v, %v", i, got, err)
		}
	}
	if got, err := s.ExecuteInSession(vendedID, answer); err != nil || got != 42 {
		t.Fatalf("second site returned %v, %v", got, err)
	}

	warnings := strings.Count(buf.String(), "WARNING: dangerous synchronous ExecuteInSession")
	if warnings != 2 {
		t.Fatalf("got %d warnings, want 2 (one per site):\n%s", warnings, buf.String())
	}
	if n := strings.Count(buf.String(), "dangerous_test.go:"); n != 2 {
		t.Errorf("warnings name this file %d times, want 2:\n%s", n, buf.String())
	}
}

// Test: a dangerous call panics when not allowed (R171)
func TestDangerousCallPanicsWhenNotAllowed(t *testing.T) {
	s := newRefTestServer(t, func(c *config.Config) { c.Session.AllowDangerousExecution = false })
	_, vendedID := newSession(t, s)
	ran := false

	msg := func() (msg string) {
		defer func() { msg = fmt.Sprint(recover()) }()
		s.ExecuteInSession(vendedID, func() (interface{}, error) {
			ran = true
			return nil, nil
		})
		return ""
	}()

	if !strings.Contains(msg, "dangerous_test.go:") {
		t.Fatalf("panic %q does not name the calling site", msg)
	}
	if ran {
		t.Fatal("the dangerous call ran its work before panicking")
	}
}

// Test: the non-dangerous path is silent (R169)
func TestNonDangerousPathIsSilent(t *testing.T) {
	s := newRefTestServer(t, func(c *config.Config) { c.Session.AllowDangerousExecution = false })
	_, vendedID := newSession(t, s)
	buf := captureLog(t)

	got, err := s.PrimitiveExecuteInSession(vendedID, answer, false)
	if err != nil || got != 42 {
		t.Fatalf("got %v, %v", got, err)
	}
	if strings.Contains(buf.String(), "dangerous") {
		t.Fatalf("non-dangerous call logged a warning:\n%s", buf.String())
	}
}

// Test: AllowDangerousExecution defaults to set (R172)
func TestAllowDangerousExecutionDefault(t *testing.T) {
	if !config.DefaultConfig().Session.AllowDangerousExecution {
		t.Fatal("AllowDangerousExecution is not set by default")
	}
}
