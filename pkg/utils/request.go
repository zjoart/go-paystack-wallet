package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func DecodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) (int, error) {
	if r.Header.Get("Content-Type") != "application/json" {
		return http.StatusUnsupportedMediaType, fmt.Errorf("Content-Type header is not application/json")
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&dst); err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}
