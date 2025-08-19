package main

import (
	"encoding/json"
	"net/http"
)

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
