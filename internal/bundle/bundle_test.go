// CRC: crc-Bundle.md
package bundle

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAddDirToZip_RegularFiles(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create test file structure
	htmlDir := filepath.Join(tmpDir, "html")
	if err := os.MkdirAll(htmlDir, 0755); err != nil {
		t.Fatal(err)
	}

	indexContent := []byte("<html>test</html>")
	if err := os.WriteFile(filepath.Join(htmlDir, "index.html"), indexContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Create ZIP
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	if err := addDirToZip(zipWriter, tmpDir, ""); err != nil {
		t.Fatalf("addDirToZip failed: %v", err)
	}
	zipWriter.Close()

	// Verify ZIP contents
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("failed to read ZIP: %v", err)
	}

	if len(zipReader.File) != 1 {
		t.Fatalf("expected 1 file, got %d", len(zipReader.File))
	}

	if zipReader.File[0].Name != "html/index.html" {
		t.Errorf("expected html/index.html, got %s", zipReader.File[0].Name)
	}
}

func TestAddDirToZip_RelativeSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require special permissions on Windows")
	}

	tmpDir := t.TempDir()

	// Create target file
	targetDir := filepath.Join(tmpDir, "shared")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	targetContent := []byte("shared content")
	if err := os.WriteFile(filepath.Join(targetDir, "data.txt"), targetContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Create symlink to target
	htmlDir := filepath.Join(tmpDir, "html")
	if err := os.MkdirAll(htmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	symlinkPath := filepath.Join(htmlDir, "link.txt")
	if err := os.Symlink("../shared/data.txt", symlinkPath); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	// Create ZIP
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	if err := addDirToZip(zipWriter, tmpDir, ""); err != nil {
		t.Fatalf("addDirToZip failed: %v", err)
	}
	zipWriter.Close()

	// Verify ZIP contents
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("failed to read ZIP: %v", err)
	}

	// Should have 2 files: data.txt and the symlink
	if len(zipReader.File) != 2 {
		t.Fatalf("expected 2 files, got %d", len(zipReader.File))
	}

	// Find symlink entry
	var symlinkEntry *zip.File
	for _, f := range zipReader.File {
		if f.Name == "html/link.txt" {
			symlinkEntry = f
			break
		}
	}

	if symlinkEntry == nil {
		t.Fatal("symlink entry not found in ZIP")
	}

	// Verify it's marked as symlink
	if symlinkEntry.Mode()&os.ModeSymlink == 0 {
		t.Error("symlink entry not marked as symlink")
	}

	// Verify symlink target is stored as content
	rc, err := symlinkEntry.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	content := make([]byte, 100)
	n, _ := rc.Read(content)
	if string(content[:n]) != "../shared/data.txt" {
		t.Errorf("symlink content = %q, want %q", string(content[:n]), "../shared/data.txt")
	}
}

func TestAddDirToZip_AbsoluteSymlink_Rejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require special permissions on Windows")
	}

	tmpDir := t.TempDir()
	htmlDir := filepath.Join(tmpDir, "html")
	if err := os.MkdirAll(htmlDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create absolute symlink (should be rejected)
	symlinkPath := filepath.Join(htmlDir, "link.txt")
	if err := os.Symlink("/etc/passwd", symlinkPath); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	err := addDirToZip(zipWriter, tmpDir, "")
	if err == nil {
		t.Fatal("expected error for absolute symlink, got nil")
	}
}

func TestAddDirToZip_EscapingSymlink_Rejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require special permissions on Windows")
	}

	tmpDir := t.TempDir()
	htmlDir := filepath.Join(tmpDir, "html")
	if err := os.MkdirAll(htmlDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create symlink that escapes bundle root
	symlinkPath := filepath.Join(htmlDir, "link.txt")
	if err := os.Symlink("../../etc/passwd", symlinkPath); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	err := addDirToZip(zipWriter, tmpDir, "")
	if err == nil {
		t.Fatal("expected error for escaping symlink, got nil")
	}
}

func TestExtractBundle_WithSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require special permissions on Windows")
	}

	// Create source directory with symlink
	srcDir := t.TempDir()
	targetDir := filepath.Join(srcDir, "shared")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	targetContent := []byte("shared content")
	if err := os.WriteFile(filepath.Join(targetDir, "data.txt"), targetContent, 0644); err != nil {
		t.Fatal(err)
	}

	htmlDir := filepath.Join(srcDir, "html")
	if err := os.MkdirAll(htmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	symlinkPath := filepath.Join(htmlDir, "link.txt")
	if err := os.Symlink("../shared/data.txt", symlinkPath); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	// Create ZIP in memory
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	if err := addDirToZip(zipWriter, srcDir, ""); err != nil {
		t.Fatal(err)
	}
	zipWriter.Close()

	// Extract to new directory
	extractDir := t.TempDir()
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range zipReader.File {
		if err := extractZipFile(f, extractDir); err != nil {
			t.Fatalf("extractZipFile(%s) failed: %v", f.Name, err)
		}
	}

	// Verify symlink was recreated
	extractedLink := filepath.Join(extractDir, "html", "link.txt")
	info, err := os.Lstat(extractedLink)
	if err != nil {
		t.Fatalf("failed to stat extracted symlink: %v", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("extracted file is not a symlink")
	}

	// Verify symlink target
	target, err := os.Readlink(extractedLink)
	if err != nil {
		t.Fatal(err)
	}

	// Normalize for comparison
	expected := filepath.FromSlash("../shared/data.txt")
	if target != expected {
		t.Errorf("symlink target = %q, want %q", target, expected)
	}

	// Verify symlink resolves correctly
	content, err := os.ReadFile(extractedLink)
	if err != nil {
		t.Fatalf("failed to read through symlink: %v", err)
	}
	if string(content) != "shared content" {
		t.Errorf("symlink content = %q, want %q", string(content), "shared content")
	}
}

func TestExtractZipFile_EscapingSymlink_Rejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require special permissions on Windows")
	}

	// Create a ZIP with a malicious symlink (crafted manually)
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	header := &zip.FileHeader{
		Name:   "html/evil.txt",
		Method: zip.Store,
	}
	header.SetMode(os.ModeSymlink | 0777)

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	// Write escaping target
	writer.Write([]byte("../../etc/passwd"))
	zipWriter.Close()

	// Try to extract
	extractDir := t.TempDir()
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}

	err = extractZipFile(zipReader.File[0], extractDir)
	if err == nil {
		t.Fatal("expected error for escaping symlink, got nil")
	}
}

func TestListFilesWithInfo_ReturnsSymlinkMetadata(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require special permissions on Windows")
	}

	// Create a ZIP with a regular file and a symlink
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Add regular file
	regularWriter, err := zipWriter.Create("html/index.html")
	if err != nil {
		t.Fatal(err)
	}
	regularWriter.Write([]byte("<html>test</html>"))

	// Add symlink
	symlinkHeader := &zip.FileHeader{
		Name:   "html/link.txt",
		Method: zip.Store,
	}
	symlinkHeader.SetMode(os.ModeSymlink | 0777)
	symlinkWriter, err := zipWriter.CreateHeader(symlinkHeader)
	if err != nil {
		t.Fatal(err)
	}
	symlinkWriter.Write([]byte("../shared/data.txt"))

	zipWriter.Close()

	// Use listFilesWithInfoFromReader to test without requiring bundled binary
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}

	files := listFilesWithInfoFromReader(zipReader)

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	// Find entries
	var regularFile, symlinkFile *FileInfo
	for i := range files {
		if files[i].Name == "html/index.html" {
			regularFile = &files[i]
		} else if files[i].Name == "html/link.txt" {
			symlinkFile = &files[i]
		}
	}

	if regularFile == nil {
		t.Fatal("regular file not found")
	}
	if regularFile.IsSymlink {
		t.Error("regular file should not be marked as symlink")
	}
	if regularFile.SymlinkTarget != "" {
		t.Errorf("regular file target = %q, want empty", regularFile.SymlinkTarget)
	}

	if symlinkFile == nil {
		t.Fatal("symlink file not found")
	}
	if !symlinkFile.IsSymlink {
		t.Error("symlink file should be marked as symlink")
	}
	if symlinkFile.SymlinkTarget != "../shared/data.txt" {
		t.Errorf("symlink target = %q, want %q", symlinkFile.SymlinkTarget, "../shared/data.txt")
	}
}

func TestValidateSymlinkTarget(t *testing.T) {
	tmpDir := t.TempDir()
	absDir, _ := filepath.Abs(tmpDir)

	tests := []struct {
		name        string
		symlinkPath string
		target      string
		wantErr     bool
	}{
		{
			name:        "valid relative symlink",
			symlinkPath: filepath.Join(tmpDir, "html", "link.txt"),
			target:      "../shared/data.txt",
			wantErr:     false,
		},
		{
			name:        "valid same-directory symlink",
			symlinkPath: filepath.Join(tmpDir, "html", "link.txt"),
			target:      "other.txt",
			wantErr:     false,
		},
		{
			name:        "escaping symlink",
			symlinkPath: filepath.Join(tmpDir, "html", "link.txt"),
			target:      "../../etc/passwd",
			wantErr:     true,
		},
		{
			name:        "deeply nested escape",
			symlinkPath: filepath.Join(tmpDir, "a", "b", "c", "link.txt"),
			target:      "../../../../outside.txt",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create parent directory for symlink path
			os.MkdirAll(filepath.Dir(tt.symlinkPath), 0755)

			err := validateSymlinkTarget(tt.symlinkPath, tt.target, absDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSymlinkTarget() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
