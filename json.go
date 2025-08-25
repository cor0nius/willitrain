package main

import (
	"encoding/json"
	"net/http"
)

// This file contains helper functions for sending standardized JSON responses.

// respondWithError logs an error message (if one is provided) and sends a
// JSON error response to the client with a given message and status code.
func (cfg *apiConfig) respondWithError(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {
		cfg.logger.Error(msg, "error", err)
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	cfg.respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

// respondWithJSON marshals a payload to JSON, sets the appropriate content-type header,
// writes the HTTP status code, and sends the JSON response to the client.
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
