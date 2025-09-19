// Full-text search implementation for conversation titles
// Based on: https://artem.krylysov.com/blog/2020/07/28/lets-build-a-full-text-search-engine/
// This package provides search functionality for finding conversations by title content
package main

import (
	"strings"
	"unicode"

	snowballeng "github.com/kljensen/snowball/english"
)

// index represents an inverted index mapping search terms to conversation IDs
// Each key is a processed token, each value is a slice of conversation IDs containing that token
type index map[string][]int

// add processes conversation titles and builds the search index
// For each title, it extracts searchable tokens and maps them to the conversation ID
func (idx index) add(titles []string) {
	for id, title := range titles {
		// Process title through text analysis pipeline to get searchable tokens
		for _, token := range analyze(title) {
			// Avoid duplicate entries for the same token in the same document
			if contains(idx[token], id) {
				continue
			}
			// Add conversation ID to the token's posting list
			idx[token] = append(idx[token], id)
		}
	}
}

// analyze processes text through the full text analysis pipeline
// This pipeline transforms raw text into normalized, searchable tokens
func analyze(text string) []string {
	tokens := tokenize(text)           // Split text into individual words
	tokens = toLower(tokens)           // Normalize to lowercase for case-insensitive search
	tokens = removeCommonWords(tokens) // Remove stop words (a, the, and, etc.)
	tokens = stem(tokens)              // Reduce words to their root forms (running -> run)
	return tokens
}

// tokenize splits text into individual words by separating on non-alphanumeric characters
// This handles punctuation, spaces, and special characters as word boundaries
func tokenize(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		// Split on any character that isn't a letter or number
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}

// toLower normalizes all tokens to lowercase for case-insensitive searching
// This ensures "Hello" and "hello" are treated as the same term
func toLower(tokens []string) []string {
	r := make([]string, len(tokens))
	for i, token := range tokens {
		r[i] = strings.ToLower(token)
	}
	return r
}

// stopWords defines common English words that should be excluded from search indexing
// These words are too common to provide meaningful search discrimination
var stopWords = map[string]struct{}{
	"a":    {},
	"and":  {},
	"be":   {},
	"have": {},
	"i":    {},
	"in":   {},
	"of":   {},
	"that": {},
	"the":  {},
	"to":   {},
}

// removeCommonWords filters out stop words from the token list
// This reduces index size and improves search quality by removing noise words
func removeCommonWords(tokens []string) []string {
	r := make([]string, 0, len(tokens))
	for _, token := range tokens {
		// Only keep tokens that are not in the stop words list
		if _, ok := stopWords[token]; !ok {
			r = append(r, token)
		}
	}
	return r
}

// stem reduces words to their root forms using the Snowball English stemmer
// This allows searches for "running" to match documents containing "run", "runs", etc.
func stem(tokens []string) []string {
	r := make([]string, len(tokens))
	for i, token := range tokens {
		// Use Porter stemming algorithm to find word roots
		r[i] = snowballeng.Stem(token, false)
	}
	return r
}

// contains checks if a conversation ID already exists in a token's posting list
// This prevents duplicate entries in the search index
func contains(slice []int, val int) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// intersection finds common conversation IDs between two sorted lists
// This implements the AND operation for multi-term search queries
// Both input slices must be sorted for this algorithm to work correctly
func intersection(a, b []int) []int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	r := make([]int, 0, maxLen)
	var i, j int
	// Two-pointer technique to find common elements in sorted arrays
	for i < len(a) && j < len(b) {
		if a[i] < b[j] {
			i++ // Advance pointer in first array
		} else if a[i] > b[j] {
			j++ // Advance pointer in second array
		} else {
			// Found common element
			r = append(r, a[i])
			i++
			j++
		}
	}
	return r
}

// search finds conversations matching the given search query
// Returns conversation IDs that contain ALL search terms (AND operation)
func (idx index) search(text string) []int {
	var r []int
	// Process search query through same analysis pipeline as indexed content
	for _, token := range analyze(text) {
		if ids, ok := idx[token]; ok {
			if r == nil {
				// First token - initialize results with its posting list
				r = ids
			} else {
				// Subsequent tokens - find intersection (documents containing ALL terms)
				r = intersection(r, ids)
			}
		} else {
			// Token not found in index - no results possible
			return nil
		}
	}
	return r
}
