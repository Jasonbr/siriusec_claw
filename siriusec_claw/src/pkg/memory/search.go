package memory

import (
	"math"
	"path/filepath"
	"strings"
	"time"
)

// HybridSearchConfig configures the hybrid search behavior.
type HybridSearchConfig struct {
	// VectorWeight is the weight for vector similarity (0-1). Default 0.7.
	VectorWeight float64 `json:"vectorWeight"`
	// TextWeight is the weight for BM25 text search (0-1). Default 0.3.
	TextWeight float64 `json:"textWeight"`
	// HalfLifeDays controls time-decay: score halves every this many days. Default 14.
	HalfLifeDays float64 `json:"halfLifeDays"`
	// MaxResults limits the number of search results. Default 10.
	MaxResults int `json:"maxResults"`
}

// DefaultHybridSearchConfig returns sensible defaults.
func DefaultHybridSearchConfig() HybridSearchConfig {
	return HybridSearchConfig{
		VectorWeight: 0.7,
		TextWeight:   0.3,
		HalfLifeDays: 14,
		MaxResults:   10,
	}
}

// MarkdownSearchResult is a search result from the Markdown store.
type MarkdownSearchResult struct {
	Filename string  `json:"filename"`
	Title    string  `json:"title"`
	Content  string  `json:"content"`
	Score    float64 `json:"score"`
	AgeDays  float64 `json:"ageDays"`
}

// SearchMarkdown performs BM25-style search over Markdown memory files
// with time-decay scoring.
func SearchMarkdown(mdStore *MarkdownStore, query string, cfg HybridSearchConfig) []MarkdownSearchResult {
	if cfg.MaxResults <= 0 {
		cfg.MaxResults = 10
	}
	if cfg.HalfLifeDays <= 0 {
		cfg.HalfLifeDays = 14
	}

	// Load all markdown content
	allContent, err := mdStore.AllMarkdownContent()
	if err != nil || len(allContent) == 0 {
		return nil
	}

	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}

	// Calculate average document length for BM25
	totalLen := 0
	for _, content := range allContent {
		totalLen += len(strings.Fields(content))
	}
	avgDL := float64(totalLen) / float64(len(allContent))

	N := float64(len(allContent))
	k1 := 1.2 // BM25 parameter
	b := 0.75 // BM25 parameter

	var results []MarkdownSearchResult

	for filename, content := range allContent {
		// BM25 scoring
		words := strings.Fields(strings.ToLower(content))
		dl := float64(len(words))

		// Create term frequency map
		tfMap := make(map[string]int)
		for _, w := range words {
			w = strings.Trim(w, ".,;:!?\"'()[]{}#@")
			if len(w) >= 2 {
				tfMap[w]++
			}
		}

		var bm25Score float64
		for _, term := range terms {
			tf := float64(tfMap[term])

			// Document frequency (how many docs contain this term)
			df := 0.0
			for _, otherContent := range allContent {
				if strings.Contains(strings.ToLower(otherContent), term) {
					df++
				}
			}

			// IDF component
			idf := math.Log((N - df + 0.5) / (df + 0.5))
			if idf < 0 {
				idf = 0
			}

			// TF component with BM25 normalization
			tfNorm := (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/avgDL))

			// Also boost title matches (first line after #)
			titleBoost := 0.0
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "#") {
					titleLower := strings.ToLower(line)
					if strings.Contains(titleLower, term) {
						titleBoost += 2.0
					}
				}
			}

			bm25Score += idf*tfNorm + titleBoost
		}

		if bm25Score <= 0 {
			continue
		}

		// Time decay
		ageDays := getFileAgeDays(filename)
		decay := 1.0
		if isDailyLogFile(filename) && cfg.HalfLifeDays > 0 {
			// Only decay daily log files, not MEMORY.md or other permanent files
			decay = math.Pow(0.5, ageDays/cfg.HalfLifeDays)
		}

		finalScore := bm25Score * decay

		// Extract title from content
		title := extractMarkdownTitle(content)

		results = append(results, MarkdownSearchResult{
			Filename: filename,
			Title:    title,
			Content:  truncateContent(content, 500),
			Score:    finalScore,
			AgeDays:  ageDays,
		})
	}

	// Sort by score descending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if len(results) > cfg.MaxResults {
		results = results[:cfg.MaxResults]
	}

	return results
}

// isDailyLogFile checks if the filename matches YYYY-MM-DD.md pattern.
func isDailyLogFile(filename string) bool {
	base := strings.TrimSuffix(filepath.Base(filename), ".md")
	_, err := time.Parse("2006-01-02", base)
	return err == nil
}

// getFileAgeDays returns how many days old a daily log file is.
func getFileAgeDays(filename string) float64 {
	base := strings.TrimSuffix(filepath.Base(filename), ".md")
	t, err := time.Parse("2006-01-02", base)
	if err != nil {
		return 0 // Non-date files don't decay
	}
	return time.Since(t).Hours() / 24.0
}

// extractMarkdownTitle gets the first heading from markdown content.
func extractMarkdownTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

// truncateContent limits content to maxChars.
func truncateContent(content string, maxChars int) string {
	if len(content) <= maxChars {
		return content
	}
	return content[:maxChars] + "..."
}
