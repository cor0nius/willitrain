package main

import (
	"encoding/json"
	"net/http"
)

// This file provides centralized helper functions for creating and sending
// standardized JSON responses. Using these helpers ensures that all API responses
// are consistent in structure, which simplifies client-side development.

// respondWithError standardizes error responses. It logs the actual error for
// server-side debugging while sending a clean, structured JSON error message to the
// client. This prevents exposing internal implementation details in error messages.
func (cfg *apiConfig) respondWithError(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {
		cfg.logger.Error(msg, "error", err)
	}
	cfg.respondWithJSON(w, code, ErrorResponse{
		Error: msg,
	})
}

// respondWithJSON handles the serialization and transmission of all successful JSON
// responses. It ensures that the correct HTTP status code and `Content-Type`
// header are set, providing a consistent and reliable response format.
func (cfg *apiConfig) respondWithJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(payload)
	if err != nil {
		cfg.logger.Error("error marshalling JSON", "error", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	_, err = w.Write(data)
	if err != nil {
		cfg.logger.Error("error writing response", "error", err)
	}
}
