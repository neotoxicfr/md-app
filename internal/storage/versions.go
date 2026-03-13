package storage

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Version represents a snapshot of a file at a point in time.
type Version struct {
	ID        string    `json:"id"`
	FileID    string    `json:"file_id"`
	Hash      string    `json:"hash"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	Message   string    `json:"message"`
}

// VersionWithContent includes the stored markdown content.
type VersionWithContent struct {
	Version
	Content string `json:"content"`
}

// versionsDir returns the directory for a file's version snapshots.
func (s *Storage) versionsDir(fileID string) (string, error) {
	if !validID(fileID) {
		return "", ErrInvalidID
	}
	return filepath.Join(s.root, ".versions", fileID), nil
}

// versionContentPath returns the path to a version's content file.
func (s *Storage) versionContentPath(fileID, versionID string) (string, error) {
	if !validID(fileID) || !validID(versionID) {
		return "", ErrInvalidID
	}
	return filepath.Join(s.root, ".versions", fileID, versionID+".md"), nil
}

// versionMetaPath returns the path to a version's metadata sidecar.
func (s *Storage) versionMetaPath(fileID, versionID string) (string, error) {
	if !validID(fileID) || !validID(versionID) {
		return "", ErrInvalidID
	}
	return filepath.Join(s.root, ".versions", fileID, versionID+".json"), nil
}

// SaveVersion creates a new version snapshot for a file.
func (s *Storage) SaveVersion(fileID, content, message string) (Version, error) {
	if !validID(fileID) {
		return Version{}, ErrInvalidID
	}
	// Verify the file exists.
	if _, err := s.GetMeta(fileID); err != nil {
		return Version{}, err
	}

	dir, err := s.versionsDir(fileID)
	if err != nil {
		return Version{}, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Version{}, fmt.Errorf("create versions dir: %w", err)
	}

	vid := uuid.New().String()
	h := sha256.Sum256([]byte(content))

	v := Version{
		ID:        vid,
		FileID:    fileID,
		Hash:      fmt.Sprintf("%x", h),
		Size:      int64(len(content)),
		CreatedAt: time.Now().UTC(),
		Message:   message,
	}

	// Write content.
	vcp, err := s.versionContentPath(fileID, vid)
	if err != nil {
		return Version{}, err
	}
	if err := os.WriteFile(vcp, []byte(content), 0644); err != nil {
		return Version{}, fmt.Errorf("write version content: %w", err)
	}

	// Write metadata sidecar.
	meta, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return Version{}, err
	}
	vmp, err := s.versionMetaPath(fileID, vid)
	if err != nil {
		return Version{}, err
	}
	if err := os.WriteFile(vmp, meta, 0644); err != nil {
		return Version{}, fmt.Errorf("write version meta: %w", err)
	}

	return v, nil
}

// ListVersions returns all versions for a file, newest first.
func (s *Storage) ListVersions(fileID string) ([]Version, error) {
	if !validID(fileID) {
		return nil, ErrInvalidID
	}
	dir, err := s.versionsDir(fileID)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Version{}, nil
		}
		return nil, err
	}

	var versions []Version
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			slog.Warn("skipping unreadable version file", "file", e.Name(), "error", err)
			continue
		}
		var v Version
		if err := json.Unmarshal(data, &v); err != nil {
			slog.Warn("skipping corrupted version metadata", "file", e.Name(), "error", err)
			continue
		}
		versions = append(versions, v)
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].CreatedAt.After(versions[j].CreatedAt)
	})
	return versions, nil
}

// GetVersion returns a specific version's content.
func (s *Storage) GetVersion(fileID, versionID string) (VersionWithContent, error) {
	vmp, err := s.versionMetaPath(fileID, versionID)
	if err != nil {
		return VersionWithContent{}, err
	}
	metaData, err := os.ReadFile(vmp)
	if err != nil {
		if os.IsNotExist(err) {
			return VersionWithContent{}, ErrNotFound
		}
		return VersionWithContent{}, err
	}

	var v Version
	if err := json.Unmarshal(metaData, &v); err != nil {
		return VersionWithContent{}, err
	}

	vcp, err := s.versionContentPath(fileID, versionID)
	if err != nil {
		return VersionWithContent{}, err
	}
	content, err := os.ReadFile(vcp)
	if err != nil {
		if os.IsNotExist(err) {
			return VersionWithContent{}, ErrNotFound
		}
		return VersionWithContent{}, err
	}

	return VersionWithContent{Version: v, Content: string(content)}, nil
}

// RestoreVersion restores a file to a specific version. It saves the current
// content as a backup version, updates the file, and records the restore as
// another version snapshot.
func (s *Storage) RestoreVersion(fileID, versionID string) (File, error) {
	if !validID(fileID) || !validID(versionID) {
		return File{}, ErrInvalidID
	}
	vc, err := s.GetVersion(fileID, versionID)
	if err != nil {
		return File{}, fmt.Errorf("get version: %w", err)
	}

	meta, err := s.GetMeta(fileID)
	if err != nil {
		return File{}, err
	}

	// Save current content as a backup version before restoring.
	current, err := s.GetContent(fileID)
	if err == nil {
		if _, err := s.SaveVersion(fileID, current.Content, "auto-save before restore"); err != nil {
			slog.Warn("auto-version before restore failed", "file_id", fileID, "error", err)
		}
	}

	// Update the file with the restored content.
	updated, err := s.Update(fileID, meta.Name, vc.Content)
	if err != nil {
		return File{}, fmt.Errorf("restore update: %w", err)
	}

	// Record the restore as a version.
	if _, err := s.SaveVersion(fileID, vc.Content, fmt.Sprintf("restored from version %s", versionID)); err != nil {
		slog.Warn("version snapshot after restore failed", "file_id", fileID, "error", err)
	}

	return updated, nil
}
