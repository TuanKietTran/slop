package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nats-io/nats.go"
)

// NATSPublisher wraps a NATS connection for publishing tasks.
type NATSPublisher struct {
	nc *nats.Conn
}

// NewNATSPublisher creates a new publisher. Falls back gracefully if NATS is
// unavailable so the HTTP server still starts in development.
func NewNATSPublisher() (*NATSPublisher, error) {
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = nats.DefaultURL
	}
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return &NATSPublisher{nc: nc}, nil
}

// Publish serialises v as JSON and publishes it to subject.
func (p *NATSPublisher) Publish(subject string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return p.nc.Publish(subject, b)
}

// Close drains and closes the underlying connection.
func (p *NATSPublisher) Close() {
	_ = p.nc.Drain()
}
