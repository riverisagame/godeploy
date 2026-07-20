package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// RespondJSON encodes data to JSON and writes it to the response writer.
// It also logs any errors encountered during the encoding process.
func RespondJSON(w http.ResponseWriter, data interface{}) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v\n", err)
	}
}
