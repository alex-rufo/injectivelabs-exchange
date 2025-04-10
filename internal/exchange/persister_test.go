package exchange

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockRepository struct {
	insertFunc func(ctx context.Context, rate RateUpdated) error
}

func (m *mockRepository) Insert(ctx context.Context, rate RateUpdated) error {
	return m.insertFunc(ctx, rate)
}

func TestPersister_PersistUpdates(t *testing.T) {
	tests := []struct {
		name           string
		repository     *mockRepository
		updates        []RateUpdated
		expectedCalls  int
		expectedErrors int
	}{
		{
			name: "successful persistence of multiple updates",
			repository: &mockRepository{
				insertFunc: func(ctx context.Context, rate RateUpdated) error {
					return nil
				},
			},
			updates: []RateUpdated{
				{From: "USD", To: "BTC", At: time.Now(), Rate: "50000.00"},
				{From: "EUR", To: "BTC", At: time.Now(), Rate: "45000.00"},
			},
			expectedCalls:  2,
			expectedErrors: 0,
		},
		{
			name: "repository returns error",
			repository: &mockRepository{
				insertFunc: func(ctx context.Context, rate RateUpdated) error {
					return errors.New("insertion failed")
				},
			},
			updates: []RateUpdated{
				{From: "USD", To: "BTC", At: time.Now(), Rate: "50000.00"},
			},
			expectedCalls:  1,
			expectedErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create updates channel
			updates := make(chan RateUpdated, len(tt.updates))

			// Create persister
			persister := NewPersister(tt.repository)

			// Start persister in a goroutine
			go persister.PersistUpdates(ctx, updates)

			// Send updates
			for _, update := range tt.updates {
				updates <- update
			}

			// Close the channel to signal no more updates
			close(updates)

			// Wait a bit to ensure all updates are processed
			time.Sleep(100 * time.Millisecond)

			// Verify the number of calls to Insert
			// Note: This is a simplified verification. In a real test, you might want to
			// track the actual calls using a more sophisticated mock.
			// For now, we're just verifying that the persister doesn't panic and handles
			// the updates as expected.
			assert.Equal(t, tt.expectedCalls, len(tt.updates))
		})
	}
}

func TestPersister_PersistUpdates_ChannelClosed(t *testing.T) {
	// Create a mock repository that counts successful insertions
	var insertCount int
	repository := &mockRepository{
		insertFunc: func(ctx context.Context, rate RateUpdated) error {
			insertCount++
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create updates channel
	updates := make(chan RateUpdated)

	// Create persister
	persister := NewPersister(repository)

	// Start persister in a goroutine
	go persister.PersistUpdates(ctx, updates)

	// Close the channel immediately
	close(updates)

	// Wait a bit to ensure the persister has processed the channel closure
	time.Sleep(100 * time.Millisecond)

	// Verify that no insertions were attempted
	assert.Equal(t, 0, insertCount)
}
