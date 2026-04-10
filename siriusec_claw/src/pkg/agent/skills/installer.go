package skills

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// InstallResult is returned after a successful skill installation.
type InstallResult struct {
	Name     string    `json:"name"`
	SkillDir string    `json:"skillDir"`
	Source   string    `json:"source"`
	Metadata *Metadata `json:"metadata,omitempty"`
}

// BinStatus reports whether a required binary is available.
type BinStatus struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Path      string `json:"path,omitempty"`
}

// ManagedSkillsDir returns the managed skills directory path.
func ManagedSkillsDir(env func(string) string) string {
	stateDir := ""
	if env != nil {
		stateDir = env("SIRIUSEC_CLAW_STATE_DIR")
	}
	if stateDir == "" {
		home := ""
		if env != nil {
			home = env("HOME")
		}
		if home == "" {
			home, _ = os.UserHomeDir()
		}
		stateDir = filepath.Join(home, ".siriusec_claw")
	}
	return filepath.Join(stateDir, "skills")
}

var githubRepoRe = regexp.MustCompile(`github\.com/([^/]+)/([^/]+)`)

// InstallFromGitHub downloads and installs a skill from a GitHub repository.
func InstallFromGitHub(ctx context.Context, repoURL string, env func(string) string) (*InstallResult, error) {
	matches := githubRepoRe.FindStringSubmatch(repoURL)
	if len(matches) < 3 {
		return nil, fmt.Errorf("invalid GitHub URL: %s (expected github.com/owner/repo)", repoURL)
	}
	owner, repo := matches[1], matches[2]
	// Clean repo name (remove .git suffix)
	repo = strings.TrimSuffix(repo, ".git")

	// Extract subpath if present (e.g., github.com/owner/repo/tree/main/path/to/skill)
	subPath := ""
	treeParts := strings.SplitN(repoURL, "/tree/", 2)
	if len(treeParts) == 2 {
		// Remove branch name (first segment after tree/)
		branchAndPath := treeParts[1]
		slashIdx := strings.Index(branchAndPath, "/")
		if slashIdx >= 0 {
			subPath = branchAndPath[slashIdx+1:]
		}
	}

	// Download zipball
	zipURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball", owner, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", zipURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read zip: %w", err)
	}

	// Extract zip to temp dir
	tmpDir, err := os.MkdirTemp("", "skill-install-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extractZip(zipData, tmpDir); err != nil {
		return nil, fmt.Errorf("extract zip: %w", err)
	}

	// Find SKILL.md in the extracted contents
	skillDir, err := findSkillDir(tmpDir, subPath)
	if err != nil {
		return nil, err
	}

	// Read and parse SKILL.md
	skillMD, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		return nil, fmt.Errorf("read SKILL.md: %w", err)
	}
	meta, _ := ParseFrontmatter(skillMD)

	// Determine skill name
	name := repo
	if subPath != "" {
		name = filepath.Base(subPath)
	}
	if meta != nil && meta.Name != "" {
		name = meta.Name
	}
	name = SanitizeName(name)

	// Copy to managed skills dir
	managedDir := ManagedSkillsDir(env)
	destDir := filepath.Join(managedDir, name)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("create skill dir: %w", err)
	}

	// Copy all files from skill source dir to dest
	if err := copyDir(skillDir, destDir); err != nil {
		os.RemoveAll(destDir)
		return nil, fmt.Errorf("copy skill files: %w", err)
	}

	// Write manifest
	manifest := &InstallManifest{
		Name:      name,
		Source:    "github",
		SourceURL: repoURL,
	}
	if err := WriteManifest(destDir, manifest); err != nil {
		os.RemoveAll(destDir)
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	return &InstallResult{
		Name:     name,
		SkillDir: destDir,
		Source:   "github",
		Metadata: meta,
	}, nil
}

// InstallFromUpload installs a skill from uploaded content.
func InstallFromUpload(name string, skillMD []byte, env func(string) string) (*InstallResult, error) {
	if len(skillMD) == 0 {
		return nil, fmt.Errorf("SKILL.md content is empty")
	}
	if len(skillMD) > 1024*1024 {
		return nil, fmt.Errorf("SKILL.md exceeds 1MB limit")
	}

	meta, _ := ParseFrontmatter(skillMD)
	if name == "" {
		if meta != nil && meta.Name != "" {
			name = meta.Name
		} else {
			name = "uploaded-skill"
		}
	}
	name = SanitizeName(name)

	managedDir := ManagedSkillsDir(env)
	destDir := filepath.Join(managedDir, name)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("create skill dir: %w", err)
	}

	if err := os.WriteFile(filepath.Join(destDir, "SKILL.md"), skillMD, 0644); err != nil {
		os.RemoveAll(destDir)
		return nil, fmt.Errorf("write SKILL.md: %w", err)
	}

	manifest := &InstallManifest{
		Name:   name,
		Source: "upload",
	}
	if err := WriteManifest(destDir, manifest); err != nil {
		os.RemoveAll(destDir)
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	return &InstallResult{
		Name:     name,
		SkillDir: destDir,
		Source:   "upload",
		Metadata: meta,
	}, nil
}

// InstallFromZip installs a skill from a zip file containing SKILL.md.
// The zip file can contain either a SKILL.md at root or a folder containing SKILL.md.
func InstallFromZip(name string, zipData []byte, env func(string) string) (*InstallResult, error) {
	if len(zipData) == 0 {
		return nil, fmt.Errorf("zip content is empty")
	}
	if len(zipData) > 10*1024*1024 {
		return nil, fmt.Errorf("zip file exceeds 10MB limit")
	}

	// Extract zip to temp dir
	tmpDir, err := os.MkdirTemp("", "skill-zip-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extractZipFlat(zipData, tmpDir); err != nil {
		return nil, fmt.Errorf("extract zip: %w", err)
	}

	// Find SKILL.md in the extracted contents
	skillDir, err := findSkillDirFlat(tmpDir)
	if err != nil {
		return nil, err
	}

	// Read and parse SKILL.md
	skillMD, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		return nil, fmt.Errorf("read SKILL.md: %w", err)
	}
	meta, _ := ParseFrontmatter(skillMD)

	// Determine skill name
	if name == "" {
		if meta != nil && meta.Name != "" {
			name = meta.Name
		} else {
			name = "uploaded-skill"
		}
	}
	name = SanitizeName(name)

	// Copy to managed skills dir
	managedDir := ManagedSkillsDir(env)
	destDir := filepath.Join(managedDir, name)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("create skill dir: %w", err)
	}

	// Copy all files from skill source dir to dest
	if err := copyDir(skillDir, destDir); err != nil {
		os.RemoveAll(destDir)
		return nil, fmt.Errorf("copy skill files: %w", err)
	}

	// Write manifest
	manifest := &InstallManifest{
		Name:   name,
		Source: "upload",
	}
	if err := WriteManifest(destDir, manifest); err != nil {
		os.RemoveAll(destDir)
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	return &InstallResult{
		Name:     name,
		SkillDir: destDir,
		Source:   "upload",
		Metadata: meta,
	}, nil
}

// DeleteSkill removes a managed skill.
func DeleteSkill(name string, env func(string) string) error {
	name = SanitizeName(name)
	managedDir := ManagedSkillsDir(env)
	skillDir := filepath.Join(managedDir, name)

	// Safety: verify the path is under managedDir
	absSkill, _ := filepath.Abs(skillDir)
	absManaged, _ := filepath.Abs(managedDir)
	if !strings.HasPrefix(absSkill, absManaged+string(filepath.Separator)) {
		return fmt.Errorf("invalid skill path")
	}

	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill '%s' not found", name)
	}

	return os.RemoveAll(skillDir)
}

// UpdateSkill re-installs a managed skill from its original source.
func UpdateSkill(ctx context.Context, name string, env func(string) string) (*InstallResult, error) {
	name = SanitizeName(name)
	managedDir := ManagedSkillsDir(env)
	skillDir := filepath.Join(managedDir, name)

	manifest, err := ReadManifest(skillDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read manifest: %w (skill may have been uploaded manually)", err)
	}

	if manifest.SourceURL == "" {
		return nil, fmt.Errorf("skill '%s' has no source URL (uploaded manually), cannot auto-update", name)
	}

	// Delete and re-install
	if err := os.RemoveAll(skillDir); err != nil {
		return nil, fmt.Errorf("remove old skill: %w", err)
	}

	return InstallFromGitHub(ctx, manifest.SourceURL, env)
}

// CheckBins checks if required binaries are available on PATH.
func CheckBins(requires *Requires) []BinStatus {
	if requires == nil || len(requires.Bins) == 0 {
		return nil
	}
	var results []BinStatus
	for _, bin := range requires.Bins {
		status := BinStatus{Name: bin}
		path, err := exec.LookPath(bin)
		if err == nil {
			status.Available = true
			status.Path = path
		}
		results = append(results, status)
	}
	return results
}

// --- helpers ---

func extractZip(data []byte, destDir string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// Strip first directory (GitHub adds owner-repo-hash/ prefix)
		name := f.Name
		slashIdx := strings.Index(name, "/")
		if slashIdx >= 0 {
			name = name[slashIdx+1:]
		}
		if name == "" {
			continue
		}

		destPath := filepath.Join(destDir, name)
		// Safety check
		if !strings.HasPrefix(destPath, destDir) {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile(destPath, data, f.Mode()); err != nil {
			return err
		}
	}
	return nil
}

func findSkillDir(rootDir, subPath string) (string, error) {
	// If subPath specified, look there directly
	if subPath != "" {
		candidate := filepath.Join(rootDir, subPath)
		if _, err := os.Stat(filepath.Join(candidate, "SKILL.md")); err == nil {
			return candidate, nil
		}
	}

	// Search for SKILL.md in root
	if _, err := os.Stat(filepath.Join(rootDir, "SKILL.md")); err == nil {
		return rootDir, nil
	}

	// Search in subdirectories (one level)
	entries, _ := os.ReadDir(rootDir)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		candidate := filepath.Join(rootDir, e.Name())
		if _, err := os.Stat(filepath.Join(candidate, "SKILL.md")); err == nil {
			return candidate, nil
		}
		// Check skills/ subdir
		skillsDir := filepath.Join(candidate, "skills")
		subEntries, _ := os.ReadDir(skillsDir)
		for _, se := range subEntries {
			if se.IsDir() {
				c := filepath.Join(skillsDir, se.Name())
				if _, err := os.Stat(filepath.Join(c, "SKILL.md")); err == nil {
					return c, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no SKILL.md found in repository")
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		destPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0644)
	})
}

// extractZipFlat extracts a zip file without stripping any prefix.
func extractZipFlat(data []byte, destDir string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		name := f.Name
		// Clean path to prevent path traversal
		name = filepath.Clean(name)
		if strings.HasPrefix(name, "..") {
			continue
		}

		destPath := filepath.Join(destDir, name)
		// Safety check
		if !strings.HasPrefix(destPath, destDir+string(filepath.Separator)) && destPath != destDir {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		fileData, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile(destPath, fileData, f.Mode()); err != nil {
			return err
		}
	}
	return nil
}

// findSkillDirFlat searches for SKILL.md in extracted zip contents.
func findSkillDirFlat(rootDir string) (string, error) {
	// Check for SKILL.md at root
	if _, err := os.Stat(filepath.Join(rootDir, "SKILL.md")); err == nil {
		return rootDir, nil
	}

	// Walk through all subdirectories to find SKILL.md
	var found string
	filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == "SKILL.md" {
			found = filepath.Dir(path)
			return filepath.SkipAll
		}
		return nil
	})

	if found != "" {
		return found, nil
	}

	return "", fmt.Errorf("no SKILL.md found in zip file")
}
