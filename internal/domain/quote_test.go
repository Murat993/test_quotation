package domain

import (
	"errors"
	"testing"
)

func TestNormalizePair(t *testing.T) {
	tests := []struct {
		name    string
		pair    string
		want    string
		wantErr error
	}{
		{
			name:    "Valid EUR/MXN",
			pair:    "EUR/MXN",
			want:    "EUR/MXN",
			wantErr: nil,
		},
		{
			name:    "Valid USD/EUR",
			pair:    "USD/EUR",
			want:    "USD/EUR",
			wantErr: nil,
		},
		{
			name:    "Valid RUB/USD",
			pair:    "RUB/USD",
			want:    "RUB/USD",
			wantErr: nil,
		},
		{
			name:    "Invalid format - no slash",
			pair:    "EURMXN",
			want:    "",
			wantErr: ErrInvalidPair,
		},
		{
			name:    "Invalid length",
			pair:    "EURO/MXN",
			want:    "",
			wantErr: ErrInvalidPair,
		},
		{
			name:    "Trim and Upper",
			pair:    " usd / eur ",
			want:    "USD/EUR",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizePair(tt.pair)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NormalizePair() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("NormalizePair() unexpected error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizePair() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewQuoteRequest(t *testing.T) {
	t.Run("Valid pair", func(t *testing.T) {
		req, err := NewQuoteRequest("EUR/USD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.Pair != "EUR/USD" {
			t.Errorf("expected pair EUR/USD, got %s", req.Pair)
		}
		if req.Status != StatusPending {
			t.Errorf("expected status %s, got %s", StatusPending, req.Status)
		}
		if req.ID == "" {
			t.Error("expected non-empty ID")
		}
	})

	t.Run("Invalid pair", func(t *testing.T) {
		_, err := NewQuoteRequest("INVALID")
		if !errors.Is(err, ErrInvalidPair) {
			t.Errorf("expected error %v, got %v", ErrInvalidPair, err)
		}
	})
}
