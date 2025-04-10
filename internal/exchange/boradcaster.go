package exchange

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alex-rufo/exchange/pkg/syncx"
)

type RateUpdated struct {
	From string    `json:"from"`
	To   string    `json:"to"`
	At   time.Time `json:"at"`
	Rate string    `json:"rate"`
}

type Broadcaster struct {
	updates               chan RateUpdated
	subcriptionBufferSize int
	subscriptions         *syncx.Map[string, chan RateUpdated]
}

func NewBroadcaster(updates chan RateUpdated, subcriptionBufferSize int) *Broadcaster {
	return &Broadcaster{
		updates:               updates,
		subcriptionBufferSize: subcriptionBufferSize,
		subscriptions:         new(syncx.Map[string, chan RateUpdated]),
	}
}

func (b *Broadcaster) ListenAndServer(ctx context.Context) {
	for {
		select {
		case update, ok := <-b.updates:
			if !ok {
				// Channel closed, stopping consumer
				return
			}

			b.subscriptions.Range(func(id string, subscription chan RateUpdated) bool {
				select {
				case subscription <- update:
					// subscription received the rate update successfully.
				default:
					// subscription is full, skip that update as we don't want to block other subscriptions.
					log.Printf("rate update '%v' skipped for subscription '%v' as channel was full", update, id)
				}

				return true
			})
		}
	}
}

func (b *Broadcaster) Close() {
	b.subscriptions.Range(func(id string, subscription chan RateUpdated) bool {
		b.Unsubscribe(id)
		return true
	})
}

func (b *Broadcaster) Subscribe(id string) (<-chan RateUpdated, error) {
	subscription := make(chan RateUpdated, b.subcriptionBufferSize)

	_, loaded := b.subscriptions.LoadOrStore(id, subscription)
	if loaded {
		return nil, fmt.Errorf("there is another subscription with the same id (%s), it can not be added", id)
	}

	return subscription, nil
}

func (b *Broadcaster) Unsubscribe(id string) {
	subscription, loaded := b.subscriptions.LoadAndDelete(id)
	if !loaded {
		return
	}

	close(subscription)
}
