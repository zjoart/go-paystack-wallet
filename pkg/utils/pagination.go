package utils

import (
	"net/http"
	"strconv"
)

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func GetPaginationDetails(r *http.Request) (int, int, int) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
		limit = val
	}
	if limit > 100 {
		limit = 100
	}

	pageStr := r.URL.Query().Get("page")
	page := 1
	if val, err := strconv.Atoi(pageStr); err == nil && val > 0 {
		page = val
	}

	offset := (page - 1) * limit
	return limit, offset, page
}
