package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alex-rufo/exchange/internal/exchange"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestServer_Start(t *testing.T) {
	subscriber := &MockSubscriber{}
	repository := &MockRepository{}
	server := NewServer(subscriber, repository)

	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleRateUpdates(w, r)
	}))
	defer ts.Close()

	// Test WebSocket connection
	wsURL := "ws" + ts.URL[4:] + "/rates"
	_, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
}

func TestServer_Close(t *testing.T) {
	subscriber := &MockSubscriber{}
	repository := &MockRepository{}
	server := NewServer(subscriber, repository)

	// Test closing without starting
	server.Close() // Should not panic

	// Test closing after starting
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleRateUpdates(w, r)
	}))
	defer ts.Close()
	server.Close() // Should not panic
}

func TestServer_handleRateUpdates_WithHistoricalData(t *testing.T) {
	subscriber := &MockSubscriber{}
	repository := &MockRepository{}
	server := NewServer(subscriber, repository)

	// Mock historical data
	expectedRates := []exchange.RateUpdated{
		{From: "USD", To: "BTC", At: time.Now(), Rate: "50000.00"},
		{From: "EUR", To: "BTC", At: time.Now(), Rate: "45000.00"},
	}
	repository.On("ListSince", mock.Anything, mock.Anything).Return(expectedRates, nil)

	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleRateUpdates(w, r)
	}))
	defer ts.Close()

	// Test WebSocket connection with since parameter
	since := time.Now().Add(-1 * time.Hour).Unix()
	wsURL := fmt.Sprintf("ws%s/rates?since=%d", ts.URL[4:], since)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// Read messages and verify
	for _, expectedRate := range expectedRates {
		_, message, err := conn.ReadMessage()
		assert.NoError(t, err)

		var receivedRate exchange.RateUpdated
		err = json.Unmarshal(message, &receivedRate)
		assert.NoError(t, err)
		assert.Equal(t, expectedRate.From, receivedRate.From)
		assert.Equal(t, expectedRate.To, receivedRate.To)
		assert.Equal(t, expectedRate.Rate, receivedRate.Rate)
	}

	repository.AssertExpectations(t)
}

func TestServer_handleRateUpdates_WithSubscription(t *testing.T) {
	subscriber := &MockSubscriber{}
	repository := &MockRepository{}
	server := NewServer(subscriber, repository)

	// Create a channel for rate updates
	rateChan := make(chan exchange.RateUpdated, 1)
	expectedRate := exchange.RateUpdated{From: "USD", To: "BTC", At: time.Now(), Rate: "50000.00"}
	rateChan <- expectedRate
	close(rateChan)

	// Mock subscription
	subscriber.On("Subscribe", mock.Anything).Return(rateChan, nil)
	subscriber.On("Unsubscribe", mock.Anything).Return()

	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleRateUpdates(w, r)
	}))
	defer ts.Close()

	// Test WebSocket connection
	wsURL := fmt.Sprintf("ws%s/rates", ts.URL[4:])
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// Read message and verify
	_, message, err := conn.ReadMessage()
	assert.NoError(t, err)

	var receivedRate exchange.RateUpdated
	err = json.Unmarshal(message, &receivedRate)
	assert.NoError(t, err)
	assert.Equal(t, expectedRate.From, receivedRate.From)
	assert.Equal(t, expectedRate.To, receivedRate.To)
	assert.Equal(t, expectedRate.Rate, receivedRate.Rate)

	subscriber.AssertExpectations(t)
}

func TestServer_handleRateUpdates_SubscriptionError(t *testing.T) {
	subscriber := &MockSubscriber{}
	repository := &MockRepository{}
	server := NewServer(subscriber, repository)

	// Mock subscription error
	subscriber.On("Subscribe", mock.Anything).Return(nil, assert.AnError)

	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleRateUpdates(w, r)
	}))
	defer ts.Close()

	// Test WebSocket connection
	wsURL := fmt.Sprintf("ws%s/rates", ts.URL[4:])
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// Connection should be closed due to subscription error
	_, _, err = conn.ReadMessage()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "websocket: close 1006")

	subscriber.AssertExpectations(t)
}

func TestServer_handleRateUpdates_HistoricalDataError(t *testing.T) {
	subscriber := &MockSubscriber{}
	repository := &MockRepository{}
	server := NewServer(subscriber, repository)

	// Mock historical data error
	repository.On("ListSince", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleRateUpdates(w, r)
	}))
	defer ts.Close()

	// Test WebSocket connection with since parameter
	since := time.Now().Add(-1 * time.Hour).Unix()
	wsURL := fmt.Sprintf("ws%s/rates?since=%d", ts.URL[4:], since)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// Connection should be closed due to historical data error
	_, _, err = conn.ReadMessage()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "websocket: close 1006")

	repository.AssertExpectations(t)
}

// MockSubscriber implements the Subscriber interface for testing
type MockSubscriber struct {
	mock.Mock
}

func (m *MockSubscriber) Subscribe(id string) (<-chan exchange.RateUpdated, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(chan exchange.RateUpdated), args.Error(1)
}

func (m *MockSubscriber) Unsubscribe(id string) {
	m.Called(id)
}

// MockRepository implements the Repository interface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) ListSince(ctx context.Context, since time.Time) ([]exchange.RateUpdated, error) {
	args := m.Called(ctx, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]exchange.RateUpdated), args.Error(1)
}
