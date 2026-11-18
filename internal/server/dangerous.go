package server

import (
	"fmt"
	"runtime"
)

// PrimitiveExecuteInSession runs fn in the session's context and waits for it.
// With dangerous false it is exactly the old ExecuteInSession. With dangerous
// true it first looks at the site that called ExecuteInSession: when
// AllowDangerousExecution is set it logs a warning the first time that site is
// seen and continues; otherwise it panics before queueing anything.
//
// dangerous is meant to be set only by ExecuteInSession, whose caller is the
// site reported. Code reviewed as never running on an executor calls this with
// dangerous false.
// CRC: crc-Server.md | Seq: seq-session-ref.md#2.2 | R169, R170, R171
func (s *Server) PrimitiveExecuteInSession(vendedID string, fn func() (interface{}, error), dangerous bool) (interface{}, error) {
	if dangerous {
		s.checkDangerous(callerSite(3))
	}
	return s.executeInSession(vendedID, fn)
}

// checkDangerous warns once per site, or panics, per AllowDangerousExecution.
// CRC: crc-Server.md | Seq: seq-session-ref.md#2.2.2 | R170, R171
func (s *Server) checkDangerous(site string) {
	if !s.config.Session.AllowDangerousExecution {
		// Seq: seq-session-ref.md#2.2.3
		panic(fmt.Sprintf("dangerous synchronous ExecuteInSession from %s (set session.allow_dangerous_execution to allow it)", site))
	}
	s.dangerousMu.Lock()
	seen := s.dangerousSites[site]
	if !seen {
		if s.dangerousSites == nil {
			s.dangerousSites = make(map[string]bool)
		}
		s.dangerousSites[site] = true
	}
	s.dangerousMu.Unlock()
	if seen {
		return
	}
	s.config.Log(0, "WARNING: dangerous synchronous ExecuteInSession from %s; it deadlocks if called on an executor", site)
}

// callerSite names the file and line of the frame skip levels up from callerSite
// itself (0 is callerSite, 1 its caller, and so on).
// Seq: seq-session-ref.md#2.2.1
func callerSite(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown site"
	}
	return fmt.Sprintf("%s:%d", file, line)
}
