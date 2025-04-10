package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/alex-rufo/exchange/internal/exchange"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Subscriber interface {
	Subscribe(id string) (<-chan exchange.RateUpdated, error)
	Unsubscribe(id string)
}

type Repository interface {
	ListSince(ctx context.Context, since time.Time) ([]exchange.RateUpdated, error)
}

var upgrader = websocket.Upgrader{
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all connections by default; customize in production!
		return true
	},
}

type Server struct {
	server     *http.Server
	subscriber Subscriber
	repository Repository
}

func NewServer(subscriber Subscriber, repository Repository) *Server {
	return &Server{
		subscriber: subscriber,
		repository: repository,
	}
}

func (s *Server) Start(port int) error {
	log.Printf("Server running on port %d\n", port)

	s.server = &http.Server{
		Addr: fmt.Sprintf(":%d", port),
	}

	http.HandleFunc("/rates", s.handleRateUpdates)
	return s.server.ListenAndServe()
}

func (s *Server) Close() {
	if s.server == nil {
		return
	}

	if err := s.server.Close(); err != nil {
		log.Printf("HTTP close error: %v", err)
	}
}

func (s *Server) handleRateUpdates(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP request to a WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Println("WebSocket client connected")

	if param := r.URL.Query().Get("since"); param != "" {
		i, err := strconv.ParseInt(param, 10, 64)
		if err != nil {
			log.Printf("Invalid since param: %v", err)
			return
		}

		since := time.Unix(i, 0)
		rates, err := s.repository.ListSince(r.Context(), since)
		if err != nil {
			log.Printf("Failed to get oild rates updated: %v", err)
			return
		}

		for _, rate := range rates {
			if err := s.writeToWS(conn, rate); err != nil {
				log.Printf("Failed to send rate udpate to the websocket: %v", err)
			}
		}
	}

	subscriptionID := uuid.NewString()
	rates, err := s.subscriber.Subscribe(subscriptionID)
	if err != nil {
		log.Printf("Subscription failed: %v", err)
		return
	}
	defer s.subscriber.Unsubscribe(subscriptionID)

	for {
		select {
		case rate, ok := <-rates:
			if !ok {
				// Rates channel was closed, we won't receive any more updates
				return
			}

			if err := s.writeToWS(conn, rate); err != nil {
				// We failed to write to the WS, let's stop the subscription.
				// TODO: we should be more careful as not all the errors mean disconnection but I wanted to keep it simple for now.
				log.Printf("Failed to send rate udpate to the websocket: %v", err)
				return
			}
		}
	}

}

func (s *Server) writeToWS(conn *websocket.Conn, rate exchange.RateUpdated) error {
	payload, err := json.Marshal(rate)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, payload)
}
