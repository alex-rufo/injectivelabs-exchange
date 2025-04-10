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

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var data FetchBitcoinPriceResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return &data, nil
}
