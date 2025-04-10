package coindesk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type FetchBitcoinPriceResponse struct {
	Time struct {
		Updated    string    `json:"updated"`
		UpdatedISO time.Time `json:"updatedISO"`
		UpdatedUK  string    `json:"updateduk"`
	} `json:"time"`
	Disclaimer string `json:"disclaimer"`
	ChartName  string `json:"chartName"`
	BPI        map[string]struct {
		Code        string  `json:"code"`
		Symbol      string  `json:"symbol"`
		Rate        string  `json:"rate"`
		Description string  `json:"description"`
		RateFloat   float64 `json:"rate_float"`
	} `json:"bpi"`
}

// Client is a structure in charge of executing API calls against CoinDesk.
// Currently it only supports one endpoint:
//   - /v1/bpi/currentprice.json
//
// TODO: it would be great to add retrials with exponential backoff.
type Client struct {
	baseURL string
	client  *http.Client
	timeout time.Duration
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) FetchBitcoinPrice(ctx context.Context) (*FetchBitcoinPriceResponse, error) {
	url := fmt.Sprintf("%s/v1/bpi/currentprice.json", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data at %s: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, payload: %v", resp.StatusCode, body)
	}

	var data FetchBitcoinPriceResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v, payload: %v", err, body)
	}

	return &data, nil
}
