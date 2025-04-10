package exchange

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInMemoryRepository_Insert(t *testing.T) {
	repo := NewInMemoryRepository(3)
	ctx := context.Background()

	// Test inserting a single rate
	rate1 := RateUpdated{
		From: "USD",
		To:   "EUR",
		At:   time.Now(),
		Rate: "1.0",
	}
	err := repo.Insert(ctx, rate1)
	assert.NoError(t, err)

	// Test inserting multiple rates
	rate2 := RateUpdated{
		From: "USD",
		To:   "EUR",
		At:   time.Now().Add(time.Hour),
		Rate: "2.0",
	}
	err = repo.Insert(ctx, rate2)
	assert.NoError(t, err)

	rate3 := RateUpdated{
		From: "USD",
		To:   "EUR",
		At:   time.Now().Add(2 * time.Hour),
		Rate: "3.0",
	}
	err = repo.Insert(ctx, rate3)
	assert.NoError(t, err)

	// Test overwriting old rates
	rate4 := RateUpdated{
		From: "USD",
		To:   "EUR",
		At:   time.Now().Add(3 * time.Hour),
		Rate: "4.0",
	}
	err = repo.Insert(ctx, rate4)
	assert.NoError(t, err)
}

func TestInMemoryRepository_ListSince(t *testing.T) {
	repo := NewInMemoryRepository(5)
	ctx := context.Background()
	now := time.Now()

	// Insert rates with different timestamps
	rates := []RateUpdated{
		{
			From: "USD",
			To:   "EUR",
			At:   now.Add(-2 * time.Hour),
			Rate: "1.0",
		},
		{
			From: "USD",
			To:   "EUR",
			At:   now.Add(-1 * time.Hour),
			Rate: "2.0",
		},
		{
			From: "USD",
			To:   "EUR",
			At:   now,
			Rate: "3.0",
		},
	}

	for _, rate := range rates {
		err := repo.Insert(ctx, rate)
		assert.NoError(t, err)
	}

	tests := []struct {
		name     string
		since    time.Time
		expected []RateUpdated
	}{
		{
			name:     "all rates after since",
			since:    now.Add(-3 * time.Hour),
			expected: rates,
		},
		// {
		// 	name:     "some rates after since",
		// 	since:    now.Add(-90 * time.Minute),
		// 	expected: rates[1:],
		// },
		// {
		// 	name:     "no rates after since",
		// 	since:    now.Add(time.Hour),
		// 	expected: []RateUpdated{},
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.ListSince(ctx, tt.since)
			assert.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(result))

			// Compare each rate
			for i, rate := range result {
				assert.Equal(t, tt.expected[i].Rate, rate.Rate)
				assert.Equal(t, tt.expected[i].From, rate.From)
				assert.Equal(t, tt.expected[i].To, rate.To)
				assert.True(t, tt.expected[i].At.Equal(rate.At))
			}
		})
	}
}
