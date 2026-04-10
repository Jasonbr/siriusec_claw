package memory

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/siriusec/siriusec_claw/pkg/logging"
	"github.com/siriusec/siriusec_claw/pkg/paths"
)

var memLog = logging.Sub("memory")

// Entry represents a single knowledge/memory entry.
type Entry struct {
	ID        string                 `json:"id"`
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	Tags      []string               `json:"tags,omitempty"`
	Source    string                 `json:"source,omitempty"`   // "user", "agent", "import"
	Category  string                 `json:"category,omitempty"` // "note", "snippet", "reference", "context"
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt int64                  `json:"createdAt"`
	UpdatedAt int64                  `json:"updatedAt"`
}

// Store manages the knowledge/memory store.
type Store struct {
	mu       sync.RWMutex
	entries  map[string]*Entry
	filePath string
}

// ResolveMemoryDir returns the memory storage directory.
func ResolveMemoryDir(env func(string) string) string {
	return filepath.Join(paths.ResolveStateDir(env), "memory")
}

// ResolveMemoryStorePath returns the memory store JSON path.
func ResolveMemoryStorePath(env func(string) string) string {
	return filepath.Join(ResolveMemoryDir(env), "entries.json")
}

// NewStore creates or loads a memory store.
func NewStore(env func(string) string) (*Store, error) {
	fp := ResolveMemoryStorePath(env)
	dir := filepath.Dir(fp)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	s := &Store{
		entries:  make(map[string]*Entry),
		filePath: fp,
	}

	data, err := os.ReadFile(fp)
	if err == nil && len(data) > 0 {
		var entries map[string]*Entry
		if json.Unmarshal(data, &entries) == nil {
			s.entries = entries
		}
	}

	memLog.Info("memory store loaded entries=%d", len(s.entries))
	return s, nil
}

// Add creates a new memory entry.
func (s *Store) Add(entry *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	now := time.Now().UnixMilli()
	entry.CreatedAt = now
	entry.UpdatedAt = now
	s.entries[entry.ID] = entry
	return s.save()
}

// Update modifies an existing entry.
func (s *Store) Update(entry *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[entry.ID]; !ok {
		return nil
	}
	entry.UpdatedAt = time.Now().UnixMilli()
	s.entries[entry.ID] = entry
	return s.save()
}

// Delete removes an entry by ID.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, id)
	return s.save()
}

// Get retrieves an entry by ID.
func (s *Store) Get(id string) *Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entries[id]
}

// List returns all entries, optionally filtered by category.
func (s *Store) List(category string) []*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Entry, 0, len(s.entries))
	for _, e := range s.entries {
		if category != "" && e.Category != category {
			continue
		}
		result = append(result, e)
	}
	return result
}

// SearchResult wraps an entry with a relevance score.
type SearchResult struct {
	Entry *Entry  `json:"entry"`
	Score float64 `json:"score"`
}

// Search finds entries matching the query using TF-based relevance scoring.
func (s *Store) Search(query string, limit int) []SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if query == "" || len(s.entries) == 0 {
		return nil
	}

	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}

	var results []SearchResult

	for _, entry := range s.entries {
		score := scoreEntry(entry, terms)
		if score > 0 {
			results = append(results, SearchResult{Entry: entry, Score: score})
		}
	}

	// Sort by score descending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// Count returns the number of entries.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0600)
}

// scoreEntry computes relevance score for an entry against query terms.
// Weights: title (3x), tags (2x), content (1x).
func scoreEntry(entry *Entry, terms []string) float64 {
	titleLower := strings.ToLower(entry.Title)
	contentLower := strings.ToLower(entry.Content)
	tagsLower := strings.ToLower(strings.Join(entry.Tags, " "))

	var score float64

	for _, term := range terms {
		titleCount := float64(strings.Count(titleLower, term))
		tagCount := float64(strings.Count(tagsLower, term))
		contentCount := float64(strings.Count(contentLower, term))

		// Weighted TF scoring
		score += titleCount * 3.0
		score += tagCount * 2.0
		score += contentCount * 1.0
	}

	// Normalize by content length to avoid bias toward long entries
	totalLen := float64(len(entry.Title) + len(entry.Content) + len(strings.Join(entry.Tags, " ")))
	if totalLen > 0 {
		score = score / math.Log2(totalLen+2)
	}

	return score
}

// tokenize splits a query string into lowercase terms.
func tokenize(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	var result []string
	seen := make(map[string]bool)
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?\"'()[]{}#@")
		if len(w) >= 2 && !seen[w] {
			seen[w] = true
			result = append(result, w)
		}
	}
	return result
}
