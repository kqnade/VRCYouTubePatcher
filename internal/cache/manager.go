package cache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"vrcvideocacher/pkg/models"
)

var (
	ErrEntryNotFound = errors.New("cache entry not found")
	ErrInvalidEntry  = errors.New("invalid cache entry")
)

// Manager handles cache directory management
type Manager struct {
	mu           sync.RWMutex
	cachePath    string
	entries      map[string]*models.CacheEntry
	maxSizeBytes int64
}

// NewManager creates a new cache manager
func NewManager(cachePath string, maxSizeGB float64) *Manager {
	maxSizeBytes := int64(maxSizeGB * 1024 * 1024 * 1024)

	// Create cache directory if it doesn't exist
	os.MkdirAll(cachePath, 0755)

	manager := &Manager{
		cachePath:    cachePath,
		entries:      make(map[string]*models.CacheEntry),
		maxSizeBytes: maxSizeBytes,
	}

	// Scan existing cache files
	manager.Scan()

	return manager
}

// AddEntry adds a new cache entry
func (m *Manager) AddEntry(id, filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	filePath := filepath.Join(m.cachePath, filename)

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	entry := &models.CacheEntry{
		ID:         id,
		FileName:   filename,
		Size:       info.Size(),
		LastAccess: time.Now(),
		Created:    info.ModTime(),
	}

	m.entries[id] = entry

	// Check if we need to evict
	m.evictIfNeeded()

	return nil
}

// GetEntry retrieves a cache entry by ID
func (m *Manager) GetEntry(id string) (*models.CacheEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.entries[id]
	if !ok {
		return nil, ErrEntryNotFound
	}

	// Return a copy
	entryCopy := *entry
	return &entryCopy, nil
}

// DeleteEntry removes a cache entry and its file
func (m *Manager) DeleteEntry(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entries[id]
	if !ok {
		return ErrEntryNotFound
	}

	// Delete file
	filePath := filepath.Join(m.cachePath, entry.FileName)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Remove from map
	delete(m.entries, id)

	return nil
}

// ListEntries returns all cache entries
func (m *Manager) ListEntries() []*models.CacheEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries := make([]*models.CacheEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		entryCopy := *entry
		entries = append(entries, &entryCopy)
	}

	// Sort by last access (most recent first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastAccess.After(entries[j].LastAccess)
	})

	return entries
}

// GetSize returns the total size of all cached files
func (m *Manager) GetSize() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total int64
	for _, entry := range m.entries {
		total += entry.Size
	}

	return total
}

// Clear removes all cache entries
func (m *Manager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id := range m.entries {
		entry := m.entries[id]
		filePath := filepath.Join(m.cachePath, entry.FileName)
		os.Remove(filePath) // Ignore errors
		delete(m.entries, id)
	}

	return nil
}

// Scan scans the cache directory and builds the entry map
func (m *Manager) Scan() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := os.ReadDir(m.cachePath)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		// Only index video files (mp4, webm)
		ext := strings.ToLower(filepath.Ext(filename))
		if ext != ".mp4" && ext != ".webm" {
			continue
		}

		// Extract video ID from filename (e.g., VIDEO_ID.mp4 -> VIDEO_ID)
		id := strings.TrimSuffix(filename, ext)

		// Get file info
		filePath := filepath.Join(m.cachePath, filename)
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		cacheEntry := &models.CacheEntry{
			ID:         id,
			FileName:   filename,
			Size:       info.Size(),
			LastAccess: info.ModTime(),
			Created:    info.ModTime(),
		}

		m.entries[id] = cacheEntry
	}

	// Evict if needed
	m.evictIfNeeded()

	return nil
}

// UpdateLastAccess updates the last access time for an entry
func (m *Manager) UpdateLastAccess(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entries[id]
	if !ok {
		return ErrEntryNotFound
	}

	entry.LastAccess = time.Now()

	// Also touch the file
	now := time.Now()
	filePath := filepath.Join(m.cachePath, entry.FileName)
	_ = os.Chtimes(filePath, now, now) // Ignore error

	return nil
}

// GetFilePath returns the absolute file path for a cache entry
func (m *Manager) GetFilePath(id string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.entries[id]
	if !ok {
		return "", ErrEntryNotFound
	}

	return filepath.Join(m.cachePath, entry.FileName), nil
}

// GetCachePath returns the cache directory path
func (m *Manager) GetCachePath() string {
	return m.cachePath
}

// evictIfNeeded performs LRU eviction if cache size exceeds limit
// Must be called with lock held
func (m *Manager) evictIfNeeded() {
	if m.maxSizeBytes <= 0 {
		return // No size limit
	}

	// Calculate current size
	currentSize := int64(0)
	for _, entry := range m.entries {
		currentSize += entry.Size
	}

	if currentSize <= m.maxSizeBytes {
		return // Within limit
	}

	// Sort entries by last access time (oldest first)
	entries := make([]*models.CacheEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastAccess.Before(entries[j].LastAccess)
	})

	// Evict oldest entries until we're under the limit
	for _, entry := range entries {
		if currentSize <= m.maxSizeBytes {
			break
		}

		// Delete file
		filePath := filepath.Join(m.cachePath, entry.FileName)
		os.Remove(filePath) // Ignore errors

		// Remove from map
		delete(m.entries, entry.ID)
		currentSize -= entry.Size
	}
}
