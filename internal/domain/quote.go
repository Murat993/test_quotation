package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type LatestQuote struct {
	Pair      string    `json:"pair"`
	Price     float64   `json:"price"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	ErrInvalidPair = errors.New("invalid pair format, expected BASE/TARGET")
)

type QuoteStatus string

const (
	StatusPending    QuoteStatus = "pending"
	StatusProcessing QuoteStatus = "processing"
	StatusDone       QuoteStatus = "done"
	StatusFailed     QuoteStatus = "failed"
)

type QuoteRequest struct {
	ID             string      `json:"id"`
	Pair           string      `json:"pair"`
	Status         QuoteStatus `json:"status"`
	Price          float64     `json:"price"`
	Error          string      `json:"error"`
	IdempotencyKey string      `json:"idempotency_key,omitzero"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

func NewQuoteRequest(pair string, idempotencyKey string) (*QuoteRequest, error) {
	if len(idempotencyKey) > 255 {
		return nil, errors.New("idempotency key too long, max 255 characters")
	}
	normalized, err := NormalizePair(pair)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &QuoteRequest{
		ID:             uuid.New().String(),
		Pair:           normalized,
		Status:         StatusPending,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func NormalizePair(pair string) (string, error) {
	parts := strings.Split(pair, "/")
	if len(parts) != 2 {
		return "", ErrInvalidPair
	}

	base := strings.ToUpper(strings.TrimSpace(parts[0]))
	target := strings.ToUpper(strings.TrimSpace(parts[1]))

	if len(base) != 3 || len(target) != 3 {
		return "", fmt.Errorf("%w: currencies must be 3 characters long", ErrInvalidPair)
	}

	return fmt.Sprintf("%s/%s", base, target), nil
}
