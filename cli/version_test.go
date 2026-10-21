package cli

import "testing"

// A stamped Version wins over build info, so `make` builds report the tag.
func TestEngineVersionPrefersStamp(t *testing.T) {
	saved := Version
	defer func() { Version = saved }()
	Version = "v9.9.9"
	if got := EngineVersion(); got != "v9.9.9" {
		t.Fatalf("EngineVersion() = %q, want the stamped v9.9.9", got)
	}
	Version = ""
	if got := EngineVersion(); got == "" || got == "v0.1.0" {
		t.Fatalf("EngineVersion() unstamped = %q, want a build-info version", got)
	}
}
