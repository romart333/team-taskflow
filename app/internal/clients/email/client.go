// Package email is a driven adapter for the external email service.
// All calls go through a circuit breaker: when the service is down the
// breaker opens and requests fail fast instead of piling up.
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sony/gobreaker/v2"
)

// Config holds the email service endpoint and circuit breaker settings.
type Config struct {
	BaseURL        string
	RequestTimeout time.Duration
	// MaxRequests allowed through in half-open state.
	MaxRequests uint32
	// Interval to reset failure counters in closed state.
	Interval time.Duration
	// Timeout before an open breaker transitions to half-open.
	Timeout time.Duration
	// MaxFailures: consecutive failures that trip the breaker.
	MaxFailures uint32
}

type inviteMessage struct {
	To       string `json:"to"`
	TeamName string `json:"team_name"`
}

// Client implements the usecase InviteNotifier port.
type Client struct {
	httpClient *http.Client
	baseURL    string
	breaker    *gobreaker.CircuitBreaker[struct{}]
}

func NewClient(cfg Config) *Client {
	breaker := gobreaker.NewCircuitBreaker[struct{}](gobreaker.Settings{
		Name:        "email-service",
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cfg.MaxFailures
		},
	})

	return &Client{
		httpClient: &http.Client{Timeout: cfg.RequestTimeout},
		baseURL:    cfg.BaseURL,
		breaker:    breaker,
	}
}

// SendInvite posts an invite notification to the email service.
func (c *Client) SendInvite(ctx context.Context, email, teamName string) error {
	_, err := c.breaker.Execute(func() (struct{}, error) {
		return struct{}{}, c.send(ctx, inviteMessage{To: email, TeamName: teamName})
	})
	if err != nil {
		return fmt.Errorf("sending invite email: %w", err)
	}
	return nil
}

func (c *Client) send(ctx context.Context, msg inviteMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("encoding invite message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building email request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling email service: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("email service responded with status %d", resp.StatusCode)
	}
	return nil
}
