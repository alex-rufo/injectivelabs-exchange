package cmd

import (
	"time"

	"github.com/alex-rufo/exchange/cmd/server"
	"github.com/alex-rufo/exchange/internal/exchange"
	"github.com/alex-rufo/exchange/internal/exchange/coindesk"
	pkgcoindesk "github.com/alex-rufo/exchange/pkg/coindesk"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"gopkg.in/tomb.v2"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run server",
	RunE: func(cmd *cobra.Command, args []string) error {
		updatesChannel := make(chan exchange.RateUpdated)
		coindeskClient := pkgcoindesk.NewClient(coindeskBaseURL, coindeskTimeout)
		coindeskFetcher := coindesk.NewPeriodicallyFetcher(coindeskClient, toCurrencies, fetchInterval)
		repository := exchange.NewInMemoryRepository(int(repositoryTTL / fetchInterval))
		boradcaster := exchange.NewBroadcaster(updatesChannel, subscriptionBufferSize)
		server := server.NewServer(boradcaster, repository)

		t, _ := tomb.WithContext(cmd.Context())

		// We are going to persist all the updates so they can be fetched later on.
		// In order to do so, we are going to create a new subscription that,
		// instead of sending the update into a WS, will persists them into a repository.
		t.Go(func() error {
			updates, err := boradcaster.Subscribe(uuid.NewString())
			if err != nil {
				return err
			}
			persister := exchange.NewPersister(repository)
			persister.PersistUpdates(cmd.Context(), updates)
			return nil
		})

		// Listen for exchange rate updates and propage them to the multiple subscriptions.
		t.Go(func() error {
			boradcaster.ListenAndServer(cmd.Context())
			return nil
		})

		// CoinDesk provider fetcher
		t.Go(func() error {
			coindeskFetcher.Run(cmd.Context(), updatesChannel)
			return nil
		})

		// Start the HTTP server
		t.Go(func() error {
			err := server.Start(port)
			return err
		})

		// Block until tomb is dying, happens either because context is cancelled or a routine experienced an error.
		<-t.Dying()

		server.Close()
		boradcaster.Close()
		coindeskFetcher.Close()
		close(updatesChannel)

		// Wait until all the goroutines have finished
		t.Wait()
		return nil
	},
}

var (
	port                   int
	toCurrencies           []string
	fetchInterval          time.Duration
	repositoryTTL          time.Duration
	subscriptionBufferSize int
	coindeskBaseURL        string
	coindeskTimeout        time.Duration
)

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().IntVarP(&port, "port", "p", 8080, "HTTP server port (defaults to 8080)")
	serverCmd.Flags().StringSliceVarP(&toCurrencies, "currencies", "c", []string{"USD"}, "List of currencies to which we want the BTC exchange rate to (defaults to USD)")
	serverCmd.Flags().DurationVarP(&fetchInterval, "interval", "i", 5*time.Second, "Interval in which the rates are going to be refreshed (defaults to 5s)")
	serverCmd.Flags().DurationVarP(&repositoryTTL, "ttl", "", 24*time.Hour, "Time until data will be evicted from the repository (defaults to 1 hour)")
	serverCmd.Flags().IntVarP(&subscriptionBufferSize, "subscripition-buffer-size", "b", 5, "Subscription buffer size to give some time to the subscription to handle the rate updates (defaults to 5)")
	serverCmd.Flags().StringVarP(&coindeskBaseURL, "coindesk-base-url", "", "https://api.coindesk.com/", "CoinDesk base URL (defaults to https://api.coindesk.com/)")
	serverCmd.Flags().DurationVarP(&coindeskTimeout, "coindesk-timeout", "", time.Second, "CoinDesk timeout (defaults to 1s)")
}
