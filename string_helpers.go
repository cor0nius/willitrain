package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// StringTransformer defines the contract for a function that can transform a string.
type StringTransformer interface {
	TransformString(t transform.Transformer, s string) (string, int, error)
}

// defaultTransformer is the production implementation of our interface.
type defaultTransformer struct{}

// TransformString calls the actual transform.String function.
func (dt defaultTransformer) TransformString(t transform.Transformer, s string) (string, int, error) {
	return transform.String(t, s)
}

// Use a variable of the interface type. This is our "injection point".
var transformer StringTransformer = defaultTransformer{}

// This file contains helper functions for string manipulation.

// normalizeCityName takes a city name string and returns a standardized version.
// It performs two main transformations:
// 1. It removes diacritical marks (e.g., "Wrocław" becomes "Wroclaw").
// 2. It converts the string to lowercase (e.g., "Wroclaw" becomes "wroclaw").
// This is crucial for creating consistent, searchable aliases for locations.
func normalizeCityName(s string) (string, error) {
	if !utf8.ValidString(s) {
		return "", fmt.Errorf("input string is not valid UTF-8")
	}
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transformer.TransformString(t, s)
	if err != nil {
		return "", err
	}
	return strings.ToLower(result), nil
}
