package memory

import (
	"math"
)

// MMRConfig configures the Maximal Marginal Relevance reranking.
type MMRConfig struct {
	// Lambda controls the trade-off between relevance and diversity.
	// 1.0 = only relevance, 0.0 = only diversity. Default 0.5.
	Lambda float64 `json:"lambda"`
	// MaxResults is the maximum number of results to return. Default 10.
	MaxResults int `json:"maxResults"`
}

// DefaultMMRConfig returns sensible defaults.
func DefaultMMRConfig() MMRConfig {
	return MMRConfig{
		Lambda:     0.5,
		MaxResults: 10,
	}
}

// MMRResult wraps a search result with its MMR score.
type MMRResult struct {
	MarkdownSearchResult
	MMRScore float64 `json:"mmrScore"`
}

// RerankWithMMR applies Maximal Marginal Relevance to diversify search results.
// It balances relevance (original score) with diversity (similarity to already selected items).
func RerankWithMMR(results []MarkdownSearchResult, query string, cfg MMRConfig) []MMRResult {
	if cfg.MaxResults <= 0 {
		cfg.MaxResults = 10
	}
	if cfg.Lambda < 0 || cfg.Lambda > 1 {
		cfg.Lambda = 0.5
	}

	if len(results) == 0 {
		return nil
	}

	// Normalize scores to [0, 1]
	maxScore := 0.0
	for _, r := range results {
		if r.Score > maxScore {
			maxScore = r.Score
		}
	}
	if maxScore == 0 {
		maxScore = 1
	}

	normalizedResults := make([]MarkdownSearchResult, len(results))
	for i, r := range results {
		normalizedResults[i] = r
		normalizedResults[i].Score = r.Score / maxScore
	}

	// MMR selection algorithm
	selected := make([]MMRResult, 0, cfg.MaxResults)
	remaining := make([]int, len(normalizedResults))
	for i := range remaining {
		remaining[i] = i
	}

	for len(selected) < cfg.MaxResults && len(remaining) > 0 {
		bestIdx := -1
		bestMMRScore := -1.0

		for _, idx := range remaining {
			relevance := normalizedResults[idx].Score

			// Calculate max similarity to already selected items
			maxSim := 0.0
			for _, sel := range selected {
				sim := contentSimilarity(normalizedResults[idx].Content, sel.Content)
				if sim > maxSim {
					maxSim = sim
				}
			}

			// MMR formula: λ * Relevance - (1-λ) * max_sim
			mmrScore := cfg.Lambda*relevance - (1-cfg.Lambda)*maxSim

			if mmrScore > bestMMRScore {
				bestMMRScore = mmrScore
				bestIdx = idx
			}
		}

		if bestIdx < 0 {
			break
		}

		// Add best result to selected
		selected = append(selected, MMRResult{
			MarkdownSearchResult: normalizedResults[bestIdx],
			MMRScore:             bestMMRScore,
		})

		// Remove from remaining
		newRemaining := make([]int, 0, len(remaining)-1)
		for _, idx := range remaining {
			if idx != bestIdx {
				newRemaining = append(newRemaining, idx)
			}
		}
		remaining = newRemaining
	}

	return selected
}

// contentSimilarity calculates cosine similarity between two text contents.
// Uses a simple bag-of-words approach with TF weighting.
func contentSimilarity(a, b string) float64 {
	// Tokenize and create term frequency maps
	tfA := termFrequency(a)
	tfB := termFrequency(b)

	// Calculate dot product and magnitudes
	dotProduct := 0.0
	magnitudeA := 0.0
	magnitudeB := 0.0

	// Calculate magnitude for A
	for _, freq := range tfA {
		magnitudeA += float64(freq * freq)
	}
	magnitudeA = math.Sqrt(magnitudeA)

	// Calculate magnitude for B
	for _, freq := range tfB {
		magnitudeB += float64(freq * freq)
	}
	magnitudeB = math.Sqrt(magnitudeB)

	// Calculate dot product
	for term, freqA := range tfA {
		if freqB, ok := tfB[term]; ok {
			dotProduct += float64(freqA * freqB)
		}
	}

	// Cosine similarity
	if magnitudeA == 0 || magnitudeB == 0 {
		return 0
	}
	return dotProduct / (magnitudeA * magnitudeB)
}

// termFrequency creates a term frequency map from text.
func termFrequency(text string) map[string]int {
	words := tokenize(text)
	tf := make(map[string]int)
	for _, word := range words {
		tf[word]++
	}
	return tf
}

// SearchWithMMR performs BM25 search followed by MMR reranking.
func SearchWithMMR(mdStore *MarkdownStore, query string, searchCfg HybridSearchConfig, mmrCfg MMRConfig) []MMRResult {
	// First perform BM25 search
	results := SearchMarkdown(mdStore, query, searchCfg)
	if len(results) == 0 {
		return nil
	}

	// Then apply MMR reranking
	return RerankWithMMR(results, query, mmrCfg)
}
