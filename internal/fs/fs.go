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
	lookupDirs []string
}

// NewPathResolver creates a new PathResolver.
func NewPathResolver(lookupDirs []string) *PathResolver {
	if len(lookupDirs) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			// This is unlikely to fail, but if it does, it's a critical error.
			panic(fmt.Sprintf("could not get current working directory: %v", err))
		}
		return &PathResolver{lookupDirs: []string{wd}}
	}

	absDirs := make([]string, len(lookupDirs))
	for i, dir := range lookupDirs {
		abs, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		absDirs[i] = abs
	}
	return &PathResolver{lookupDirs: absDirs}
}

// Resolve finds an absolute path, assuming a new file in the first lookup
// directory if it doesn't exist.
func (r *PathResolver) Resolve(relativePath string) string {
	if existing := r.ResolveExisting(relativePath); existing != "" {
		return existing
	}
	// If not found, create the path relative to the first lookup directory.
	return filepath.Join(r.lookupDirs[0], relativePath)
}

// ResolveExisting finds an absolute path only if the file exists.
func (r *PathResolver) ResolveExisting(relativePath string) string {
	for _, dir := range r.lookupDirs {
		absPath := filepath.Join(dir, relativePath)
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
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
