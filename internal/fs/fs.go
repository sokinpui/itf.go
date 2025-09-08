package fs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"itf/internal/ui"
)

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
			ui.Warning("Invalid lookup directory '%s', ignoring: %v", dir, err)
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

// ConfirmAndCreateDirs prompts the user to create directories and creates them.
func ConfirmAndCreateDirs(dirs map[string]struct{}) bool {
	if len(dirs) == 0 {
		ui.Info("\nNo new directories need to be created.")
		return true
	}

	sortedDirs := make([]string, 0, len(dirs))
	for dir := range dirs {
		sortedDirs = append(sortedDirs, dir)
	}
	sort.Strings(sortedDirs)

	ui.Info("\nThe following directories need to be created:")
	for _, dir := range sortedDirs {
		ui.Path("- %s", dir)
	}

	fmt.Print(ui.Prompt("Do you want to create all these directories? (y/N): "))
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(response)) != "y" {
		ui.Warning("Directory creation declined. Exiting.")
		return false
	}

	ui.Info("\nCreating directories...")
	for _, dir := range sortedDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			ui.Error("Error creating directory '%s': %v", dir, err)
			ui.Error("Aborting due to directory creation failure.")
			return false
		}
		ui.Success("  -> Created: %s", dir)
	}
	return true
}
