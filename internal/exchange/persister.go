package exchange

import (
	"context"
	"log"
)

type Repository interface {
	Insert(ctx context.Context, rate RateUpdated) error
}

type Persister struct {
	repository Repository
}

func NewPersister(repository Repository) *Persister {
	return &Persister{
		repository: repository,
	}
}

func (p *Persister) PersistUpdates(ctx context.Context, updates <-chan RateUpdated) {
	for {
		select {
		case rate, ok := <-updates:
			if !ok {
				// updates channel was closed, we won't receive any more updates
				return
			}

			if err := p.repository.Insert(ctx, rate); err != nil {
				log.Println("Failed to persist the rate into the repository", err, rate)
			}
		}
	}
}
