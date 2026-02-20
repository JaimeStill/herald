package pagination

import (
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/JaimeStill/herald/pkg/query"
)

// SortFields wraps []query.SortField with flexible JSON unmarshaling.
// Accepts either a string ("name,-created_at") or an array of SortField objects.
type SortFields []query.SortField

// UnmarshalJSON supports unmarshaling from a comma-separated string or array format.
func (s *SortFields) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = query.ParseSortFields(str)
		return nil
	}

	var fields []query.SortField
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*s = fields
	return nil
}

// PageRequest represents a client request for a page of data with optional search and sorting.
type PageRequest struct {
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
	Search   *string    `json:"search,omitempty"`
	Sort     SortFields `json:"sort,omitempty"`
}

// Normalize adjusts the request to ensure valid pagination values based on the config.
func (r *PageRequest) Normalize(cfg Config) {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = cfg.DefaultPageSize
	}
	if r.PageSize > cfg.MaxPageSize {
		r.PageSize = cfg.MaxPageSize
	}
}

// Offset calculates the number of records to skip based on page and page size.
func (r *PageRequest) Offset() int {
	return (r.Page - 1) * r.PageSize
}

// PageRequestFromQuery parses pagination parameters from URL query values.
// Supported parameters: page, page_size, search, sort.
func PageRequestFromQuery(values url.Values, cfg Config) PageRequest {
	page, _ := strconv.Atoi(values.Get("page"))
	pageSize, _ := strconv.Atoi(values.Get("page_size"))

	var search *string
	if s := values.Get("search"); s != "" {
		search = &s
	}

	sort := query.ParseSortFields(values.Get("sort"))

	req := PageRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
		Sort:     sort,
	}

	req.Normalize(cfg)
	return req
}

// PageResult holds a page of data along with pagination metadata.
type PageResult[T any] struct {
	Data       []T `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalPages int `json:"total_pages"`
}

// NewPageResult creates a PageResult with calculated total pages.
func NewPageResult[T any](data []T, total, page, pageSize int) PageResult[T] {
	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}

	if data == nil {
		data = []T{}
	}

	return PageResult[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
