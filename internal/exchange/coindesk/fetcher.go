package coindesk

import (
	"context"
	"log"
	"time"

	"github.com/alex-rufo/exchange/internal/exchange"
	"github.com/alex-rufo/exchange/pkg/coindesk"
)

const (
	CurrencyBTC = "BTC"
)

type client interface {
	FetchBitcoinPrice(ctx context.Context) (*coindesk.FetchBitcoinPriceResponse, error)
}

type Fetcher struct {
	client       client
	toCurrencies []string
}

func NewFetcher(client client, toCurrencies []string) *Fetcher {
	return &Fetcher{
		client:       client,
		toCurrencies: toCurrencies,
	}
}

func (f *Fetcher) Fetch(ctx context.Context) ([]exchange.RateUpdated, error) {
	response, err := f.client.FetchBitcoinPrice(ctx)
	if err != nil {
		return nil, err
	}

	rates := []exchange.RateUpdated{}
	for _, currency := range f.toCurrencies {
		price, ok := response.BPI[currency]
		if !ok {
			log.Printf("CoinDesk rate not found for currency %s: %v\n", currency, err)
			continue
		}

		rates = append(rates, exchange.RateUpdated{
			From: currency,
			To:   CurrencyBTC,
			At:   response.Time.UpdatedISO,
			Rate: price.Rate,
		})
	}

	return rates, nil
}

// PeriodicallyFetcher check for exchange rate updates every interval period.
type PeriodicallyFetcher struct {
	fetcher  Fetcher
	interval time.Duration
	done     chan struct{}
	stopped  chan struct{}
}

func NewPeriodicallyFetcher(client client, toCurrencies []string, interval time.Duration) *PeriodicallyFetcher {
	return &PeriodicallyFetcher{
		fetcher:  *NewFetcher(client, toCurrencies),
		interval: interval,
		done:     make(chan struct{}),
		stopped:  make(chan struct{}),
	}
}

func (f *PeriodicallyFetcher) Run(ctx context.Context, output chan<- exchange.RateUpdated) {
	ticker := time.NewTicker(f.interval)
	defer ticker.Stop()
	defer func() { f.stopped <- struct{}{} }()

	for {
		select {
		case <-f.done:
			// A signal to stop fetching periodically has been received.
			return
		default:
			rates, err := f.fetcher.Fetch(ctx)
			if err != nil {
				log.Println("Error fetching", err)
				continue
			}

			for _, rate := range rates {
				select {
				case output <- rate:
					// Output received the rate update successfully, let's wait for the next tick
					<-ticker.C
				case <-ticker.C:
					// The rate could not be handled in time and the next tick arrived, let's skip that rate
				}
			}
		}
	}
}

func (f *PeriodicallyFetcher) Close() {
	close(f.done) // sends a signal to the Run function (infite loop) that it should stop.
	<-f.stopped   // the infinite loop has finished, we can consider the Close function to be completed.
}
