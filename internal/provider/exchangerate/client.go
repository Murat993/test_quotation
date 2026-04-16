package exchangerate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Provider struct {
	apiKey string
}

func NewProvider(apiKey string) *Provider {
	return &Provider{apiKey: apiKey}
}

type exchangeRateResponse struct {
	Result          string             `json:"result"`
	ConversionRates map[string]float64 `json:"conversion_rates"`
	ErrorType       string             `json:"error-type"`
}

func (p *Provider) GetRate(ctx context.Context, pair string) (float64, error) {
	parts := strings.Split(pair, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid pair format: %s", pair)
	}
	base := parts[0]
	target := parts[1]

	url := fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/latest/%s", p.apiKey, base)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("external api status: %d", resp.StatusCode)
	}

	var data exchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	if data.Result == "error" {
		return 0, fmt.Errorf("external api error: %s", data.ErrorType)
	}

	rate, ok := data.ConversionRates[target]
	if !ok {
		return 0, fmt.Errorf("rate for %s not found", target)
	}

	return rate, nil
}
