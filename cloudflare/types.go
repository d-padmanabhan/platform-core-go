package cloudflare

import "encoding/json"

// APIErrorItem represents a single error returned by Cloudflare.
type APIErrorItem struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ResultInfo contains pagination metadata for list responses.
type ResultInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
	Count      int `json:"count"`
	TotalCount int `json:"total_count"`
}

type envelope struct {
	Success    bool            `json:"success"`
	Errors     []APIErrorItem  `json:"errors"`
	Result     json.RawMessage `json:"result"`
	ResultInfo *ResultInfo     `json:"result_info,omitempty"`
}

// Zone represents a Cloudflare DNS zone.
type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
