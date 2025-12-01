package api

import (
	"can-db-writer/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// parseQueryParams parses common query parameters from HTTP request
func parseQueryParams(r *http.Request) (models.QueryParams, error) {
	params := models.QueryParams{
		Limit: 100, // default limit
	}

	// Parse start_time
	if startTimeStr := r.URL.Query().Get("start_time"); startTimeStr != "" {
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return params, fmt.Errorf("invalid start_time format: %v", err)
		}
		params.StartTime = &t
	}

	// Parse end_time
	if endTimeStr := r.URL.Query().Get("end_time"); endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return params, fmt.Errorf("invalid end_time format: %v", err)
		}
		params.EndTime = &t
	}

	// Parse can_id (supports both decimal and hex)
	if canIDStr := r.URL.Query().Get("can_id"); canIDStr != "" {
		var canID uint64
		var err error
		if len(canIDStr) > 2 && canIDStr[:2] == "0x" {
			canID, err = strconv.ParseUint(canIDStr[2:], 16, 32)
		} else {
			canID, err = strconv.ParseUint(canIDStr, 10, 32)
		}
		if err != nil {
			return params, fmt.Errorf("invalid can_id format: %v", err)
		}
		canID32 := uint32(canID)
		params.CANID = &canID32
	}

	// Parse interface
	params.Interface = r.URL.Query().Get("interface")

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return params, fmt.Errorf("invalid limit format: %v", err)
		}
		params.Limit = limit
	}

	// Parse offset
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return params, fmt.Errorf("invalid offset format: %v", err)
		}
		params.Offset = offset
	}

	return params, nil
}

// respondWithError sends an error response
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON sends a JSON response
func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to marshal response"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
