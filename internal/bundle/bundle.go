// Package bundle provides functionality for bundling site files into the binary.
// Spec: deployment.md
package bundle

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// MagicMarker identifies bundled binaries
	MagicMarker = "UISERVER"
	// FooterSize: 8 bytes magic + 8 bytes offset + 8 bytes size
	FooterSize = 24
)

var IGNORE_FILES = regexp.MustCompile(`^(|.*/)((#|\.#)[^/]*|[^/]*~)$`)

// Footer contains metadata about the bundled ZIP
type Footer struct {
	Magic  [8]byte // "UISERVER"
	Offset int64   // Offset to start of ZIP data
	Size   int64   // Size of ZIP data
}

// CreateBundle creates a new bundled binary.
// sourceBinary: path to the ui binary (can be bundled or unbundled)
// siteDir: directory containing site files
// outputPath: path for the bundled binary
func CreateBundle(sourceBinary, siteDir, outputPath string) error {
	// Get the size of the executable portion (excluding any existing bundle)
	binarySize, err := GetBinarySize(sourceBinary)
	if err != nil {
		return fmt.Errorf("failed to get binary size: %w", err)
	}

	// Open source binary
	srcFile, err := os.Open(sourceBinary)
	if err != nil {
		return fmt.Errorf("failed to open source binary: %w", err)
	}
	defer srcFile.Close()

	// Create output file
	outFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy only the executable portion (without any existing bundle)
	if _, err := io.CopyN(outFile, srcFile, binarySize); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// Create ZIP in memory
	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	// Add site files to ZIP
	if err := addDirToZip(zipWriter, siteDir, ""); err != nil {
		zipWriter.Close()
		return fmt.Errorf("failed to add files to ZIP: %w", err)
	}

	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close ZIP writer: %w", err)
	}

	zipData := zipBuf.Bytes()
	zipSize := int64(len(zipData))

	// Write ZIP data
	if _, err := outFile.Write(zipData); err != nil {
		return fmt.Errorf("failed to write ZIP data: %w", err)
	}

	// Write footer
	footer := Footer{
		Offset: binarySize,
		Size:   zipSize,
	}
	copy(footer.Magic[:], MagicMarker)

	if err := binary.Write(outFile, binary.LittleEndian, footer.Offset); err != nil {
		return fmt.Errorf("failed to write offset: %w", err)
	}
	if err := binary.Write(outFile, binary.LittleEndian, footer.Size); err != nil {
		return fmt.Errorf("failed to write size: %w", err)
	}
	if _, err := outFile.Write(footer.Magic[:]); err != nil {
		return fmt.Errorf("failed to write magic: %w", err)
	}

	return nil
}

// addDirToZip recursively adds directory contents to ZIP
func addDirToZip(zipWriter *zip.Writer, sourceDir, basePath string) error {
	return filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() || IGNORE_FILES.MatchString(filePath) {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, filePath)
		if err != nil {
			return err
		}

		// Create ZIP entry with forward slashes
		zipPath := filepath.Join(basePath, relPath)
		zipPath = filepath.ToSlash(zipPath)

		writer, err := zipWriter.Create(zipPath)
		if err != nil {
			return err
		}

		// Copy file content
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

// GetBinarySize returns the size of the executable portion (excluding bundle).
// If bundled, returns the offset to the bundle. Otherwise returns total file size.
func GetBinarySize(binaryPath string) (int64, error) {
	file, err := os.Open(binaryPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open binary: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to stat binary: %w", err)
	}

	fileSize := info.Size()
	if fileSize < FooterSize {
		return fileSize, nil
	}

	// Read footer
	if _, err := file.Seek(fileSize-FooterSize, 0); err != nil {
		return 0, fmt.Errorf("failed to seek to footer: %w", err)
	}

	var footer Footer
	if err := binary.Read(file, binary.LittleEndian, &footer.Offset); err != nil {
		return fileSize, nil
	}
	if err := binary.Read(file, binary.LittleEndian, &footer.Size); err != nil {
		return fileSize, nil
	}
	if _, err := file.Read(footer.Magic[:]); err != nil {
		return fileSize, nil
	}

	// Check magic marker
	if bytes.Equal(footer.Magic[:], []byte(MagicMarker)) {
		return footer.Offset, nil
	}

	return fileSize, nil
}

// IsBundled checks if the current binary has bundled content.
func IsBundled() (bool, error) {
	exePath, err := os.Executable()
	if err != nil {
		return false, err
	}

	file, err := os.Open(exePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return false, err
	}

	fileSize := info.Size()
	if fileSize < FooterSize {
		return false, nil
	}

	// Read footer
	if _, err := file.Seek(fileSize-FooterSize, 0); err != nil {
		return false, err
	}

	var footer Footer
	if err := binary.Read(file, binary.LittleEndian, &footer.Offset); err != nil {
		return false, nil
	}
	if err := binary.Read(file, binary.LittleEndian, &footer.Size); err != nil {
		return false, nil
	}
	if _, err := file.Read(footer.Magic[:]); err != nil {
		return false, nil
	}

	return bytes.Equal(footer.Magic[:], []byte(MagicMarker)), nil
}

// GetBundleReader returns a zip.Reader for the bundled content.
// Returns nil if the binary is not bundled.
func GetBundleReader() (*zip.Reader, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	file, err := os.Open(exePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open executable: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat executable: %w", err)
	}

	fileSize := info.Size()
	if fileSize < FooterSize {
		return nil, nil
	}

	// Read footer
	if _, err := file.Seek(fileSize-FooterSize, 0); err != nil {
		return nil, fmt.Errorf("failed to seek to footer: %w", err)
	}

	var footer Footer
	if err := binary.Read(file, binary.LittleEndian, &footer.Offset); err != nil {
		return nil, nil
	}
	if err := binary.Read(file, binary.LittleEndian, &footer.Size); err != nil {
		return nil, nil
	}
	if _, err := file.Read(footer.Magic[:]); err != nil {
		return nil, nil
	}

	// Check magic marker
	if !bytes.Equal(footer.Magic[:], []byte(MagicMarker)) {
		return nil, nil
	}

	// Read ZIP data
	if _, err := file.Seek(footer.Offset, 0); err != nil {
		return nil, fmt.Errorf("failed to seek to ZIP data: %w", err)
	}

	zipData := make([]byte, footer.Size)
	if _, err := io.ReadFull(file, zipData); err != nil {
		return nil, fmt.Errorf("failed to read ZIP data: %w", err)
	}

	// Open ZIP reader
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), footer.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP reader: %w", err)
	}

	return zipReader, nil
}

// ExtractBundle extracts bundled content to a directory.
func ExtractBundle(targetDir string) error {
	zipReader, err := GetBundleReader()
	if err != nil {
		return err
	}
	if zipReader == nil {
		return fmt.Errorf("binary is not bundled")
	}

	for _, f := range zipReader.File {
		if err := extractZipFile(f, targetDir); err != nil {
			return fmt.Errorf("failed to extract %s: %w", f.Name, err)
		}
	}

	return nil
}

// extractZipFile extracts a single file from ZIP
func extractZipFile(f *zip.File, targetDir string) error {
	targetPath := filepath.Join(targetDir, f.Name)

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	// Open ZIP file entry
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// Create target file
	outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

// ListFiles returns a list of files in the bundle.
func ListFiles() ([]string, error) {
	zipReader, err := GetBundleReader()
	if err != nil {
		return nil, err
	}
	if zipReader == nil {
		return nil, fmt.Errorf("binary is not bundled")
	}

	var files []string
	for _, f := range zipReader.File {
		files = append(files, f.Name)
	}
	return files, nil
}

// ReadFile reads a file from the bundle.
func ReadFile(name string) ([]byte, error) {
	zipReader, err := GetBundleReader()
	if err != nil {
		return nil, err
	}
	if zipReader == nil {
		return nil, fmt.Errorf("binary is not bundled")
	}

	// Clean path
	name = path.Clean(name)
	name = strings.TrimPrefix(name, "/")

	for _, f := range zipReader.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}

	return nil, fmt.Errorf("file not found: %s", name)
}

// ListFilesInDir returns files in a subdirectory of the bundle.
func ListFilesInDir(dir string) ([]string, error) {
	zipReader, err := GetBundleReader()
	if err != nil {
		return nil, err
	}
	if zipReader == nil {
		return nil, fmt.Errorf("binary is not bundled")
	}

	// Clean and normalize the directory path
	dir = path.Clean(dir)
	dir = strings.TrimPrefix(dir, "/")
	if dir != "" && !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	var files []string
	for _, f := range zipReader.File {
		if strings.HasPrefix(f.Name, dir) && !f.FileInfo().IsDir() {
			// Get filename relative to the directory
			relPath := strings.TrimPrefix(f.Name, dir)
			// Only include files directly in this directory, not subdirectories
			if !strings.Contains(relPath, "/") && relPath != "" {
				files = append(files, f.Name)
			}
		}
	}
	return files, nil
}

// ZipFileSystem implements fs.FS for serving files from a ZIP archive.
// Files are served from the html/ subdirectory within the ZIP.
type ZipFileSystem struct {
	reader *zip.Reader
	prefix string // Subdirectory prefix (e.g., "html")
}

// NewZipFileSystem creates a new ZipFileSystem from a zip.Reader.
// Files are served from the html/ subdirectory.
func NewZipFileSystem(reader *zip.Reader) *ZipFileSystem {
	return &ZipFileSystem{reader: reader, prefix: "html"}
}

// NewZipFileSystemWithPrefix creates a ZipFileSystem with a custom prefix.
func NewZipFileSystemWithPrefix(reader *zip.Reader, prefix string) *ZipFileSystem {
	return &ZipFileSystem{reader: reader, prefix: prefix}
}

// Open implements fs.FS interface.
func (zfs *ZipFileSystem) Open(name string) (fs.File, error) {
	// Clean the path and remove leading slash
	name = strings.TrimPrefix(path.Clean(name), "/")
	if name == "." {
		name = ""
	}

	// Build target path with prefix (e.g., "html/index.html")
	targetPath := name
	if zfs.prefix != "" {
		if name == "" {
			targetPath = zfs.prefix
		} else {
			targetPath = zfs.prefix + "/" + name
		}
	}

	// Find the file in ZIP
	for _, f := range zfs.reader.File {
		if f.Name == targetPath && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}

			// Read entire file content
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, err
			}

			return &zipFile{
				name:    filepath.Base(name),
				content: content,
				reader:  bytes.NewReader(content),
				info:    f.FileInfo(),
			}, nil
		}
	}

	return nil, fs.ErrNotExist
}

// zipFile implements fs.File interface
type zipFile struct {
	name    string
	content []byte
	reader  *bytes.Reader
	info    fs.FileInfo
}

func (zf *zipFile) Read(p []byte) (int, error) {
	return zf.reader.Read(p)
}

func (zf *zipFile) Seek(offset int64, whence int) (int64, error) {
	return zf.reader.Seek(offset, whence)
}

func (zf *zipFile) Close() error {
	return nil
}

func (zf *zipFile) Stat() (fs.FileInfo, error) {
	return zf.info, nil
}
