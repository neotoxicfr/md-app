package storage

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound  = errors.New("file not found")
	ErrForbidden = errors.New("access denied")
	ErrInvalidID = errors.New("invalid file ID")

	uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func validID(id string) bool {
	return uuidRe.MatchString(id)
}

// File represents a stored markdown document.
type File struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`       // filename without extension
	Slug      string    `json:"slug"`       // URL-safe name
	Path      string    `json:"path"`       // relative path (folder/subfolder)
	Size      int64     `json:"size"`
	Hash      string    `json:"hash"`       // SHA-256 of content
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FileWithContent adds the raw content to a File.
type FileWithContent struct {
	File
	Content string `json:"content"`
}

// Storage manages markdown files on disk.
type Storage struct {
	root string
}

// New creates a new Storage backed by the given root directory.
func New(root string) *Storage {
	return &Storage{root: root}
}

// metaPath returns the path to a file's JSON metadata sidecar.
func (s *Storage) metaPath(id string) (string, error) {
	if !validID(id) {
		return "", ErrInvalidID
	}
	return filepath.Join(s.root, ".meta", id+".json"), nil
}

// contentPath returns the path to a file's markdown content.
func (s *Storage) contentPath(id string) (string, error) {
	if !validID(id) {
		return "", ErrInvalidID
	}
	return filepath.Join(s.root, "files", id+".md"), nil
}

// saveMeta persists a File's metadata.
func (s *Storage) saveMeta(f File) error {
	mp, err := s.metaPath(f.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(mp), 0750); err != nil {
		return err
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(mp, b, 0600)
}

// Create stores a new markdown document and returns its metadata.
func (s *Storage) Create(name, relPath, content string) (File, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	if err := os.MkdirAll(filepath.Join(s.root, "files"), 0750); err != nil {
		return File{}, err
	}

	cp, err := s.contentPath(id)
	if err != nil {
		return File{}, err
	}
	if err := os.WriteFile(cp, []byte(content), 0600); err != nil {
		return File{}, err
	}

	h := sha256.Sum256([]byte(content))
	f := File{
		ID:        id,
		Name:      sanitizeName(name),
		Slug:      toSlug(name),
		Path:      sanitizePath(relPath),
		Size:      int64(len(content)),
		Hash:      fmt.Sprintf("%x", h),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.saveMeta(f); err != nil {
		return File{}, err
	}

	return f, nil
}

// GetContent returns the markdown content of a file.
func (s *Storage) GetContent(id string) (FileWithContent, error) {
	meta, err := s.GetMeta(id)
	if err != nil {
		return FileWithContent{}, err
	}
	cp, err := s.contentPath(id)
	if err != nil {
		return FileWithContent{}, err
	}
	b, err := os.ReadFile(cp)
	if err != nil {
		if os.IsNotExist(err) {
			return FileWithContent{}, ErrNotFound
		}
		return FileWithContent{}, err
	}
	return FileWithContent{File: meta, Content: string(b)}, nil
}

// GetMeta returns the metadata for a file.
func (s *Storage) GetMeta(id string) (File, error) {
	mp, err := s.metaPath(id)
	if err != nil {
		return File{}, err
	}
	b, err := os.ReadFile(mp)
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, ErrNotFound
		}
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return File{}, err
	}
	return f, nil
}

// Update replaces a file's content and refreshes its metadata.
func (s *Storage) Update(id, name, content string) (File, error) {
	f, err := s.GetMeta(id)
	if err != nil {
		return File{}, err
	}

	cp, err := s.contentPath(id)
	if err != nil {
		return File{}, err
	}
	if err := os.WriteFile(cp, []byte(content), 0600); err != nil {
		return File{}, err
	}

	h := sha256.Sum256([]byte(content))
	f.Name = sanitizeName(name)
	f.Slug = toSlug(name)
	f.Size = int64(len(content))
	f.Hash = fmt.Sprintf("%x", h)
	f.UpdatedAt = time.Now().UTC()

	if err := s.saveMeta(f); err != nil {
		return File{}, err
	}
	return f, nil
}

// Delete removes a file and its metadata.
func (s *Storage) Delete(id string) error {
	if _, err := s.GetMeta(id); err != nil {
		return err
	}
	cp, err := s.contentPath(id)
	if err != nil {
		return err
	}
	mp, err := s.metaPath(id)
	if err != nil {
		return err
	}
	errContent := os.Remove(cp)
	errMeta := os.Remove(mp)
	return errors.Join(errContent, errMeta)
}

// List returns all file metadata, sorted by UpdatedAt desc.
func (s *Storage) List() ([]File, error) {
	metaDir := filepath.Join(s.root, ".meta")
	entries, err := os.ReadDir(metaDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []File{}, nil
		}
		return nil, err
	}

	var files []File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		f, err := s.GetMeta(id)
		if err != nil {
			slog.Warn("skipping corrupted file metadata", "id", id, "error", err)
			continue
		}
		files = append(files, f)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].UpdatedAt.After(files[j].UpdatedAt)
	})
	return files, nil
}

// ImportReader imports content from an io.Reader (upload).
func (s *Storage) ImportReader(name string, r io.Reader) (File, error) {
	b, err := io.ReadAll(io.LimitReader(r, 50<<20)) // 50 MB max
	if err != nil {
		return File{}, err
	}
	return s.Create(name, "", string(b))
}

// ---- helpers ----

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "untitled"
	}
	// Remove path separators
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	return name
}

func sanitizePath(p string) string {
	if strings.ContainsRune(p, 0) {
		return ""
	}
	cleaned := filepath.Clean(strings.ReplaceAll(p, "\\", "/"))
	// Prevent path traversal and absolute paths
	if strings.HasPrefix(cleaned, "..") || strings.HasPrefix(cleaned, "/") {
		return ""
	}
	return cleaned
}

func toSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var sb strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			sb.WriteRune(r)
		case r == ' ' || r == '_' || r == '-':
			sb.WriteRune('-')
		}
	}
	result := strings.Trim(sb.String(), "-")
	if result == "" {
		return "untitled"
	}
	return result
}
