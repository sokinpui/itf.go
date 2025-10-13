package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// GetFileSHA256 computes the SHA256 hash of a file's content.
func GetFileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// IsEmpty checks if a directory is empty.
func IsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// PathResolver finds absolute paths for files.
type PathResolver struct {
	wd string
}

// NewPathResolver creates a new PathResolver.
func NewPathResolver() *PathResolver {
	wd, err := os.Getwd()
	if err != nil {
		// This is unlikely to fail, but if it does, it's a critical error.
		panic(fmt.Sprintf("could not get current working directory: %v", err))
	}
	return &PathResolver{wd: wd}
}

// Resolve finds an absolute path for a given relative path.
func (r *PathResolver) Resolve(relativePath string) string {
	if filepath.IsAbs(relativePath) {
		return filepath.Clean(relativePath)
	}
	return filepath.Join(r.wd, relativePath)
}

// ResolveExisting finds an absolute path only if the file exists.
func (r *PathResolver) ResolveExisting(relativePath string) string {
	path := r.Resolve(relativePath)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// GetFileActionsAndDirs determines which files are new vs. modified and
// which directories need to be created.
func GetFileActionsAndDirs(targetPaths []string) (map[string]string, map[string]struct{}) {
	fileActions := make(map[string]string)
	dirsToCreate := make(map[string]struct{})

	for _, path := range targetPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fileActions[path] = "create"
			dir := filepath.Dir(path)
			if dir != "." && dir != "/" {
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					dirsToCreate[dir] = struct{}{}
				}
			}
		} else {
			fileActions[path] = "modify"
		}
	}
	return fileActions, dirsToCreate
}

// CreateDirs creates a set of directories.
func CreateDirs(dirs map[string]struct{}) error {
	if len(dirs) == 0 {
		return nil
	}

	for dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creating directory '%s': %w", dir, err)
		}
	}
	return nil
}

// TrashFile moves a file to the trash directory, preserving its relative path.
func TrashFile(path string, trashPath string, wd string) error {
	relPath, err := filepath.Rel(wd, path)
	if err != nil {
		// Fallback for paths outside wd, though this is not expected.
		relPath = filepath.Base(path)
	}

	destPath := filepath.Join(trashPath, relPath)
	destDir := filepath.Dir(destPath)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("could not create trash subdirectory: %w", err)
	}

	if err := os.Rename(path, destPath); err != nil {
		return fmt.Errorf("could not move file to trash: %w", err)
	}

	return nil
}

// RestoreFileFromTrash moves a file from the trash back to its original location.
func RestoreFileFromTrash(originalPath string, trashPath string, wd string) error {
	relPath, err := filepath.Rel(wd, originalPath)
	if err != nil {
		relPath = filepath.Base(originalPath)
	}

	srcPath := filepath.Join(trashPath, relPath)
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found in trash: %s", srcPath)
	}

	return os.Rename(srcPath, originalPath)
}
