package main

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// This file contains helper functions for string manipulation.

// normalizeCityName takes a city name string and returns a standardized version.
// It performs two main transformations:
// 1. It removes diacritical marks (e.g., "Wrocław" becomes "Wroclaw").
// 2. It converts the string to lowercase (e.g., "Wroclaw" becomes "wroclaw").
// This is crucial for creating consistent, searchable aliases for locations.
func normalizeCityName(s string) (string, error) {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transform.String(t, s)
	if err != nil {
		return "", err
	}
	return strings.ToLower(result), nil
}
