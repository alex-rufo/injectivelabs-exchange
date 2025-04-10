package exchange

import (
	"container/ring"
	"context"
	"time"
)

type InMemoryRepository struct {
	rates *ring.Ring
}

func NewInMemoryRepository(maxSize int) *InMemoryRepository {
	return &InMemoryRepository{
		rates: ring.New(maxSize),
	}
}

// Insert adds the received RateUpdated into the ring buffer. In case the ring is full,
// the oldest rate will be overriden.
func (r *InMemoryRepository) Insert(_ context.Context, rate RateUpdated) error {
	r.rates.Value = rate
	r.rates = r.rates.Next()

	return nil
}

// ListSince returns all the RateUpdated structs that have At newer than the passed since time.
// TODO: we should add pagination as a very old since could return a lot of rates.
func (r *InMemoryRepository) ListSince(ctx context.Context, since time.Time) ([]RateUpdated, error) {
	var result []RateUpdated

	r.rates.Do(func(a any) {
		if a == nil {
			// empty ring node
			return
		}

		rate := a.(RateUpdated)
		if rate.At.After(since) {
			result = append(result, rate)
		}
	})

	return result, nil
}
