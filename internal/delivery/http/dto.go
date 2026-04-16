package http

import (
	"time"
)

type requestUpdateRequest struct {
	Pair string `json:"pair" example:"USD/EUR"`
}

type requestUpdateResponse struct {
	RequestID string `json:"request_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type quoteRequestResponse struct {
	Price     float64   `json:"price,omitzero" example:"75.5"`
	UpdatedAt time.Time `json:"updated_at"`
}

type latestQuoteResponse struct {
	Price     float64   `json:"price" example:"75.5"`
	UpdatedAt time.Time `json:"updated_at"`
}

type errorResponse struct {
	Error string `json:"error" example:"invalid request body"`
}
