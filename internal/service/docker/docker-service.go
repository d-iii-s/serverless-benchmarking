package docker

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
)

// CopyFromContainer copies a file or directory from a container to the host filesystem.
// containerPath is the path inside the container to copy from.
// destPath is the directory on the host where the files will be extracted.
func CopyFromContainer(ctx context.Context, cli *client.Client, containerID, containerPath, destPath string) error {
	reader, stat, err := cli.CopyFromContainer(ctx, containerID, containerPath)
	if err != nil {
		return fmt.Errorf("failed to copy from container: %w", err)
	}
	defer reader.Close()

	baseName := filepath.Base(containerPath)
	if stat.Name != "" {
		baseName = stat.Name
	}

	collectedDir := filepath.Join(destPath, "collected")
	if err := os.MkdirAll(collectedDir, 0755); err != nil {
		return fmt.Errorf("failed to create collected directory: %w", err)
	}

	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		cleanName := strings.TrimPrefix(header.Name, "/")
		targetPath := filepath.Join(collectedDir, cleanName)

		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(collectedDir)) {
			log.Printf("Warning: skipping potentially unsafe path: %s", header.Name)
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directories for %s: %w", targetPath, err)
			}

			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			outFile.Close()

			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				log.Printf("Warning: failed to set permissions on %s: %v", targetPath, err)
			}

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directories for symlink %s: %w", targetPath, err)
			}
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				log.Printf("Warning: failed to create symlink %s: %v", targetPath, err)
			}

		default:
			log.Printf("Warning: unsupported file type for %s (type: %c)", header.Name, header.Typeflag)
		}
	}

	log.Printf("Extracted %s to %s", baseName, collectedDir)
	return nil
}
