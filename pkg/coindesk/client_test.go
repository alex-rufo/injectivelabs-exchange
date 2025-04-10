package coindesk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchBitcoinPrice(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/bpi/currentprice.json" {
			t.Errorf("Expected to request '/v1/bpi/currentprice.json', got: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected 'GET' request, got '%s'", r.Method)
		}

		// Create a sample response
		response := FetchBitcoinPriceResponse{
			Time: struct {
				Updated    string    `json:"updated"`
				UpdatedISO time.Time `json:"updatedISO"`
				UpdatedUK  string    `json:"updateduk"`
			}{
				Updated:    "Apr 8, 2024 19:59:00 UTC",
				UpdatedISO: time.Now(),
				UpdatedUK:  "Apr 8, 2024 20:59:00 BST",
			},
			Disclaimer: "This data was produced from the CoinDesk Bitcoin Price Index",
			ChartName:  "Bitcoin",
			BPI: map[string]struct {
				Code        string  `json:"code"`
				Symbol      string  `json:"symbol"`
				Rate        string  `json:"rate"`
				Description string  `json:"description"`
				RateFloat   float64 `json:"rate_float"`
			}{
				"USD": {
					Code:        "USD",
					Symbol:      "&#36;",
					Rate:        "69,420.00",
					Description: "United States Dollar",
					RateFloat:   69420.00,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a client with the mock server URL
	client := NewClient(server.URL, time.Second)

	// Test successful response
	ctx := context.Background()
	response, err := client.FetchBitcoinPrice(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify response data
	if response.ChartName != "Bitcoin" {
		t.Errorf("Expected ChartName to be 'Bitcoin', got %s", response.ChartName)
	}

	usdRate, exists := response.BPI["USD"]
	if !exists {
		t.Error("Expected USD rate to exist in BPI")
	}
	if usdRate.RateFloat != 69420.00 {
		t.Errorf("Expected USD rate to be 69420.00, got %f", usdRate.RateFloat)
	}
}

func TestFetchBitcoinPrice_ErrorHandling(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	ctx := context.Background()

	_, err := client.FetchBitcoinPrice(ctx)
	if err == nil {
		t.Error("Expected an error for 500 status code, got nil")
	}
}
