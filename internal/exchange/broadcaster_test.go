package exchange

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscribe(t *testing.T) {
	updates := make(chan RateUpdated, 10)
	broadcaster := NewBroadcaster(updates, 5)

	// Test successful subscription
	subscription, err := broadcaster.Subscribe("test-id")
	require.NoError(t, err)
	assert.NotNil(t, subscription)

	// Test duplicate subscription
	_, err = broadcaster.Subscribe("test-id")
	assert.Error(t, err)
	assert.Equal(t, "there is another subscription with the same id (test-id), it can not be added", err.Error())
}

func TestUnsubscribe(t *testing.T) {
	updates := make(chan RateUpdated, 10)
	broadcaster := NewBroadcaster(updates, 5)

	// Subscribe first
	subscription, err := broadcaster.Subscribe("test-id")
	require.NoError(t, err)

	// Test unsubscribe
	broadcaster.Unsubscribe("test-id")

	// Verify channel is closed
	_, ok := <-subscription
	assert.False(t, ok)

	// Test unsubscribe non-existent subscription (should not panic)
	broadcaster.Unsubscribe("non-existent")
}

func TestBroadcast(t *testing.T) {
	updates := make(chan RateUpdated, 10)
	broadcaster := NewBroadcaster(updates, 5)

	// Start listener in goroutine
	go broadcaster.ListenAndServer()

	// Create two subscriptions
	sub1, err := broadcaster.Subscribe("sub1")
	require.NoError(t, err)
	sub2, err := broadcaster.Subscribe("sub2")
	require.NoError(t, err)

	// Create test update
	update := RateUpdated{
		From: "USD",
		To:   "EUR",
		At:   time.Now(),
		Rate: "1.2",
	}

	// Send update
	updates <- update

	// Verify both subscribers receive the update
	select {
	case received := <-sub1:
		assert.Equal(t, update, received)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("sub1 did not receive update")
	}

	select {
	case received := <-sub2:
		assert.Equal(t, update, received)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("sub2 did not receive update")
	}
}

func TestListenAndServer(t *testing.T) {
	updates := make(chan RateUpdated, 10)
	broadcaster := NewBroadcaster(updates, 5)

	// Create subscription
	sub, err := broadcaster.Subscribe("test-id")
	require.NoError(t, err)

	// Start listener in goroutine
	go broadcaster.ListenAndServer()

	// Create test update
	update := RateUpdated{
		From: "USD",
		To:   "EUR",
		At:   time.Now(),
		Rate: "1.2",
	}

	// Send update
	updates <- update

	// Verify subscriber receives the update
	select {
	case received := <-sub:
		assert.Equal(t, update, received)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("subscriber did not receive update")
	}

	// Test channel closure
	close(updates)
	// Wait a bit to ensure the listener has time to process the closure
	time.Sleep(100 * time.Millisecond)
}

func TestClose(t *testing.T) {
	updates := make(chan RateUpdated, 10)
	broadcaster := NewBroadcaster(updates, 5)

	// Create multiple subscriptions
	sub1, err := broadcaster.Subscribe("sub1")
	require.NoError(t, err)
	sub2, err := broadcaster.Subscribe("sub2")
	require.NoError(t, err)

	// Close all subscriptions
	broadcaster.Close()

	// Verify all channels are closed
	_, ok := <-sub1
	assert.False(t, ok)
	_, ok = <-sub2
	assert.False(t, ok)
}

func TestBroadcastWithFullChannel(t *testing.T) {
	updates := make(chan RateUpdated, 10)
	broadcaster := NewBroadcaster(updates, 1) // Small buffer size

	// Start listener in goroutine
	go broadcaster.ListenAndServer()

	// Create subscription
	sub, err := broadcaster.Subscribe("test-id")
	require.NoError(t, err)

	// Create test updates
	update1 := RateUpdated{
		From: "USD",
		To:   "EUR",
		At:   time.Now(),
		Rate: "1.2",
	}
	update2 := RateUpdated{
		From: "EUR",
		To:   "USD",
		At:   time.Now(),
		Rate: "0.8",
	}

	// Send updates through the updates channel to fill the subscription
	updates <- update1
	updates <- update2

	// Add some time make sure that update1 is not consumed yet
	time.Sleep(10 * time.Millisecond)

	// Verify first update is received
	select {
	case received := <-sub:
		assert.Equal(t, update1, received)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("did not receive first update")
	}

	// Channel should be empty now
	select {
	case <-sub:
		t.Fatal("channel should be empty")
	default:
		// Expected case
	}
}
