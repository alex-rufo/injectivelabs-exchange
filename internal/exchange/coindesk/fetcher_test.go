package coindesk

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alex-rufo/exchange/internal/exchange"
	"github.com/alex-rufo/exchange/pkg/coindesk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetcher_Fetch(t *testing.T) {
	tests := []struct {
		name          string
		client        client
		toCurrencies  []string
		expectedRates []exchange.RateUpdated
		expectedErr   error
	}{
		{
			name: "successful fetch with multiple currencies",
			client: &mockClient{
				fetchFunc: func(ctx context.Context) (*coindesk.FetchBitcoinPriceResponse, error) {
					return &coindesk.FetchBitcoinPriceResponse{
						Time: struct {
							Updated    string    `json:"updated"`
							UpdatedISO time.Time `json:"updatedISO"`
							UpdatedUK  string    `json:"updateduk"`
						}{
							Updated:    "Apr 8, 2024 19:59:00 UTC",
							UpdatedISO: time.Unix(1000, 10),
							UpdatedUK:  "Apr 8, 2024 20:59:00 BST",
						},
						BPI: map[string]struct {
							Code        string  `json:"code"`
							Symbol      string  `json:"symbol"`
							Rate        string  `json:"rate"`
							Description string  `json:"description"`
							RateFloat   float64 `json:"rate_float"`
						}{
							"USD": {
								Code:      "USD",
								Rate:      "50000.00",
								RateFloat: 50000.00,
							},
							"EUR": {
								Code:      "EUR",
								Rate:      "45000.00",
								RateFloat: 45000.00,
							},
						},
					}, nil
				},
			},
			toCurrencies: []string{"USD", "EUR"},
			expectedRates: []exchange.RateUpdated{
				{From: "USD", To: CurrencyBTC, At: time.Unix(1000, 10), Rate: "50000.00"},
				{From: "EUR", To: CurrencyBTC, At: time.Unix(1000, 10), Rate: "45000.00"},
			},
		},
		{
			name: "client returns error",
			client: &mockClient{
				fetchFunc: func(ctx context.Context) (*coindesk.FetchBitcoinPriceResponse, error) {
					return nil, errors.New("api error")
				},
			},
			toCurrencies: []string{"USD"},
			expectedErr:  errors.New("api error"),
		},
		{
			name: "currency not found in response",
			client: &mockClient{
				fetchFunc: func(ctx context.Context) (*coindesk.FetchBitcoinPriceResponse, error) {
					return &coindesk.FetchBitcoinPriceResponse{
						Time: struct {
							Updated    string    `json:"updated"`
							UpdatedISO time.Time `json:"updatedISO"`
							UpdatedUK  string    `json:"updateduk"`
						}{
							Updated:    "Apr 8, 2024 19:59:00 UTC",
							UpdatedISO: time.Unix(1000, 10),
							UpdatedUK:  "Apr 8, 2024 20:59:00 BST",
						},
						BPI: map[string]struct {
							Code        string  `json:"code"`
							Symbol      string  `json:"symbol"`
							Rate        string  `json:"rate"`
							Description string  `json:"description"`
							RateFloat   float64 `json:"rate_float"`
						}{
							"USD": {
								Code:      "USD",
								Rate:      "50000.00",
								RateFloat: 50000.00,
							},
						},
					}, nil
				},
			},
			toCurrencies:  []string{"EUR"},
			expectedRates: []exchange.RateUpdated{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := NewFetcher(tt.client, tt.toCurrencies)
			rates, err := fetcher.Fetch(context.Background())

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(t, err)
			assert.Len(t, rates, len(tt.expectedRates))

			for i, rate := range rates {
				assert.Equal(t, tt.expectedRates[i].From, rate.From)
				assert.Equal(t, tt.expectedRates[i].At, rate.At)
				assert.Equal(t, tt.expectedRates[i].To, rate.To)
				assert.Equal(t, tt.expectedRates[i].Rate, rate.Rate)
			}
		})
	}
}

func TestPeriodicallyFetcher_Run(t *testing.T) {
	tests := []struct {
		name          string
		client        client
		toCurrencies  []string
		interval      time.Duration
		ctxTimeout    time.Duration
		expectedRates []exchange.RateUpdated
	}{
		{
			name: "successful periodic fetch",
			client: &mockClient{
				fetchFunc: func(ctx context.Context) (*coindesk.FetchBitcoinPriceResponse, error) {
					return &coindesk.FetchBitcoinPriceResponse{
						Time: struct {
							Updated    string    `json:"updated"`
							UpdatedISO time.Time `json:"updatedISO"`
							UpdatedUK  string    `json:"updateduk"`
						}{
							Updated:    "Apr 8, 2024 19:59:00 UTC",
							UpdatedISO: time.Unix(1000, 10),
							UpdatedUK:  "Apr 8, 2024 20:59:00 BST",
						},
						BPI: map[string]struct {
							Code        string  `json:"code"`
							Symbol      string  `json:"symbol"`
							Rate        string  `json:"rate"`
							Description string  `json:"description"`
							RateFloat   float64 `json:"rate_float"`
						}{
							"USD": {
								Code:      "USD",
								Rate:      "50000.00",
								RateFloat: 50000.00,
							},
						},
					}, nil
				},
			},
			toCurrencies: []string{"USD"},
			interval:     10 * time.Millisecond,
			ctxTimeout:   45 * time.Millisecond,
			expectedRates: []exchange.RateUpdated{
				{From: "USD", To: CurrencyBTC, At: time.Unix(1000, 10), Rate: "50000.00"},
				{From: "USD", To: CurrencyBTC, At: time.Unix(1000, 10), Rate: "50000.00"},
				{From: "USD", To: CurrencyBTC, At: time.Unix(1000, 10), Rate: "50000.00"},
				{From: "USD", To: CurrencyBTC, At: time.Unix(1000, 10), Rate: "50000.00"},
				{From: "USD", To: CurrencyBTC, At: time.Unix(1000, 10), Rate: "50000.00"},
			},
		},
		{
			name: "client returns error during periodic fetch",
			client: &mockClient{
				fetchFunc: func(ctx context.Context) (*coindesk.FetchBitcoinPriceResponse, error) {
					return nil, errors.New("api error")
				},
			},
			toCurrencies:  []string{"USD"},
			interval:      10 * time.Millisecond,
			ctxTimeout:    11 * time.Millisecond,
			expectedRates: []exchange.RateUpdated{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
			defer cancel()

			output := make(chan exchange.RateUpdated, 10)
			fetcher := NewPeriodicallyFetcher(tt.client, tt.toCurrencies, tt.interval)

			go fetcher.Run(ctx, output)

			// Collect all rates received before context timeout
			var receivedRates []exchange.RateUpdated
			for {
				select {
				case rate := <-output:
					receivedRates = append(receivedRates, rate)
				case <-ctx.Done():
					// Compare received rates with expected rates
					assert.Len(t, receivedRates, len(tt.expectedRates))
					for i, rate := range receivedRates {
						assert.Equal(t, tt.expectedRates[i].From, rate.From)
						assert.Equal(t, tt.expectedRates[i].To, rate.To)
						assert.Equal(t, tt.expectedRates[i].At, rate.At)
						assert.Equal(t, tt.expectedRates[i].Rate, rate.Rate)
					}
					return
				}
			}
		})
	}
}

// mockClient implements the client interface for testing
type mockClient struct {
	fetchFunc func(ctx context.Context) (*coindesk.FetchBitcoinPriceResponse, error)
}

func (m *mockClient) FetchBitcoinPrice(ctx context.Context) (*coindesk.FetchBitcoinPriceResponse, error) {
	return m.fetchFunc(ctx)
}
