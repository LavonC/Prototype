package capsule

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/unicompute/unicompute/internal/protocol"
)

// Compress creates a tar.gz archive from a directory, calculating hashes along the way.
// It returns a list of FileEntry objects for the manifest and the total uncompressed size.
func Compress(sourceDir, destFile string) ([]protocol.FileEntry, int64, error) {
	out, err := os.Create(destFile)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create archive file: %w", err)
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	var entries []protocol.FileEntry
	var totalSize int64

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the destination file itself if it's inside the source dir
		if path == destFile {
			return nil
		}

		// Rel path for the archive
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Clean up the path for standard tar format
		relPath = filepath.ToSlash(relPath)
		if relPath == "." {
			return nil
		}

		// Ignore common unnecessary directories
		if info.IsDir() && (relPath == ".git" || relPath == "node_modules" || relPath == "__pycache__" || relPath == "venv") {
			return filepath.SkipDir
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		entry := protocol.FileEntry{
			Path:  relPath,
			Size:  info.Size(),
			IsDir: info.IsDir(),
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			// Tee reader to calculate hash while copying
			hasher := sha256.New()
			reader := io.TeeReader(file, hasher)

			if _, err := io.Copy(tw, reader); err != nil {
				return err
			}

			entry.SHA256 = hex.EncodeToString(hasher.Sum(nil))
			totalSize += info.Size()
		}

		entries = append(entries, entry)
		return nil
	})

	if err != nil {
		return nil, 0, fmt.Errorf("failed to walk source directory: %w", err)
	}

	return entries, totalSize, nil
}

// Extract tar.gz archive to a destination directory.
func Extract(archiveFile, destDir string) error {
	in, err := os.Open(archiveFile)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer in.Close()

	gr, err := gzip.NewReader(in)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Prevent path traversal attacks (Zip Slip)
		cleanPath := filepath.Clean(header.Name)
		if strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) || cleanPath == ".." {
			return fmt.Errorf("invalid path in archive: %s", header.Name)
		}

		target := filepath.Join(destDir, cleanPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create directory for file: %w", err)
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return fmt.Errorf("failed to write file content: %w", err)
			}
			file.Close()
		}
	}

	return nil
}
