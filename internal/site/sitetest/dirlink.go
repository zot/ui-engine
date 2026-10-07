// Package sitetest holds test helpers for building site directories.
// CRC: crc-AppDiscovery.md | Test: test-AppDiscovery.md
package sitetest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// LinkDir makes link a directory link to target: a junction on Windows
// (no privilege needed, the shape ark's activation link takes there), a
// symlink elsewhere. A relative target is taken relative to link's directory.
func LinkDir(t *testing.T, target, link string) {
	t.Helper()
	if runtime.GOOS != "windows" {
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("cannot create symlinks: %v", err)
		}
		return
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(link), target)
	}
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J %s %s: %v\n%s", link, target, err, out)
	}
}
