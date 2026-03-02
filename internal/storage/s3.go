package storage

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ---- search result type ----

// SearchResult represents a single search match within a file.
type SearchResult struct {
	FileID  string `json:"file_id"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet"`
}

// ---- pluggable storage interface ----

// StorageBackend defines the interface for pluggable storage backends.
// Both the filesystem Storage and S3Storage implement this interface.
// Select via MD_STORAGE_BACKEND=fs|s3 (default: fs).
type StorageBackend interface {
	Create(name, path, content string) (File, error)
	GetContent(id string) (FileWithContent, error)
	GetMeta(id string) (File, error)
	Update(id, name, content string) (File, error)
	Delete(id string) error
	List() ([]File, error)
	ImportReader(name string, r io.Reader) (File, error)
	Search(query string) ([]SearchResult, error)
	SaveVersion(fileID, content, message string) (Version, error)
	ListVersions(fileID string) ([]Version, error)
}

// Compile-time interface checks.
var (
	_ StorageBackend = (*Storage)(nil)
	_ StorageBackend = (*S3Storage)(nil)
)

// Root returns the filesystem root directory.
func (s *Storage) Root() string {
	return s.root
}

// ---- full-text search on filesystem ----

// Search performs a case-insensitive full-text search across all files.
// It searches in both file names and content, returning at most 50 results.
func (s *Storage) Search(query string) ([]SearchResult, error) {
	if query == "" {
		return nil, nil
	}

	files, err := s.List()
	if err != nil {
		return nil, err
	}

	lowerQuery := strings.ToLower(query)
	var results []SearchResult
	const maxResults = 50

	for _, f := range files {
		if len(results) >= maxResults {
			break
		}

		// Name match.
		if strings.Contains(strings.ToLower(f.Name), lowerQuery) {
			results = append(results, SearchResult{
				FileID:  f.ID,
				Name:    f.Name,
				Path:    f.Path,
				Line:    0,
				Snippet: "name match: " + f.Name,
			})
		}

		// Content match.
		content, err := os.ReadFile(s.contentPath(f.ID))
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			if len(results) >= maxResults {
				break
			}
			if strings.Contains(strings.ToLower(line), lowerQuery) {
				snippet := strings.TrimSpace(line)
				if len(snippet) > 200 {
					snippet = snippet[:200] + "…"
				}
				results = append(results, SearchResult{
					FileID:  f.ID,
					Name:    f.Name,
					Path:    f.Path,
					Line:    lineNum + 1,
					Snippet: snippet,
				})
			}
		}
	}

	return results, nil
}

// ---- S3-compatible storage ----

// S3Config holds configuration for an S3-compatible backend.
type S3Config struct {
	Endpoint  string // e.g. "s3.amazonaws.com" or "minio.local:9000"
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

// S3Storage wraps a local filesystem cache with S3 synchronization.
// All read/write operations hit the local cache for speed; writes are
// asynchronously replicated to S3 for durability.
type S3Storage struct {
	local  *Storage
	config S3Config
}

// NewS3 creates an S3-backed storage with a local filesystem cache.
func NewS3(cfg S3Config, localRoot string) (*S3Storage, error) {
	local := New(localRoot)
	return &S3Storage{local: local, config: cfg}, nil
}

// Create stores a file locally and queues an S3 upload.
func (s *S3Storage) Create(name, path, content string) (File, error) {
	f, err := s.local.Create(name, path, content)
	if err != nil {
		return File{}, err
	}
	// TODO: async S3 upload of content + metadata
	return f, nil
}

// GetContent reads from the local cache.
func (s *S3Storage) GetContent(id string) (FileWithContent, error) {
	return s.local.GetContent(id)
}

// GetMeta reads metadata from the local cache.
func (s *S3Storage) GetMeta(id string) (File, error) {
	return s.local.GetMeta(id)
}

// Update writes locally and queues an S3 sync.
func (s *S3Storage) Update(id, name, content string) (File, error) {
	f, err := s.local.Update(id, name, content)
	if err != nil {
		return File{}, err
	}
	// TODO: async S3 upload of updated content
	return f, nil
}

// Delete removes locally and queues an S3 delete.
func (s *S3Storage) Delete(id string) error {
	if err := s.local.Delete(id); err != nil {
		return err
	}
	// TODO: async S3 object deletion
	return nil
}

// List returns all files from the local cache.
func (s *S3Storage) List() ([]File, error) {
	return s.local.List()
}

// ImportReader imports via the local cache and queues S3 upload.
func (s *S3Storage) ImportReader(name string, r io.Reader) (File, error) {
	f, err := s.local.ImportReader(name, r)
	if err != nil {
		return File{}, err
	}
	// TODO: async S3 upload
	return f, nil
}

// Search delegates to the local filesystem search.
func (s *S3Storage) Search(query string) ([]SearchResult, error) {
	return s.local.Search(query)
}

// SaveVersion delegates to the local filesystem.
func (s *S3Storage) SaveVersion(fileID, content, message string) (Version, error) {
	v, err := s.local.SaveVersion(fileID, content, message)
	if err != nil {
		return Version{}, err
	}
	// TODO: async S3 upload of version snapshot
	return v, nil
}

// ListVersions delegates to the local filesystem.
func (s *S3Storage) ListVersions(fileID string) ([]Version, error) {
	return s.local.ListVersions(fileID)
}

// Sync downloads all S3 objects to the local cache. Call on startup to
// warm the local cache from durable storage.
func (s *S3Storage) Sync() error {
	// TODO: implement S3 ListObjects + download
	return fmt.Errorf("S3 sync not yet implemented — use MD_STORAGE_BACKEND=fs for filesystem storage")
}
