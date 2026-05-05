package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	expoSendEndpoint    = "https://exp.host/--/api/v2/push/send"
	expoReceiptsEndpoint = "https://exp.host/--/api/v2/push/getReceipts"
)

// Message represents a single Expo push notification.
type Message struct {
	To    string `json:"to"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Data  any    `json:"data,omitempty"`
	Sound string `json:"sound,omitempty"`
}

// Ticket is the per-message result returned by Expo after POST /v2/push/send.
// When Status is "ok", ID holds the ticket ID for later receipt polling.
type Ticket struct {
	Status  string         `json:"status"` // "ok" | "error"
	ID      string         `json:"id,omitempty"`
	Message string         `json:"message,omitempty"`
	Details *TicketDetails `json:"details,omitempty"`
}

// TicketDetails carries the structured error from a failed send ticket.
type TicketDetails struct {
	Error string `json:"error,omitempty"` // e.g. "DeviceNotRegistered"
}

// ticketResponse is the full Expo batch-send response body.
type ticketResponse struct {
	Data []Ticket `json:"data"`
}

// Receipt is the per-ticket result from POST /v2/push/getReceipts.
type Receipt struct {
	Status  string          `json:"status"` // "ok" | "error"
	Message string          `json:"message,omitempty"`
	Details *ReceiptDetails `json:"details,omitempty"`
}

// ReceiptDetails carries the structured error from a failed delivery receipt.
type ReceiptDetails struct {
	Error string `json:"error,omitempty"` // e.g. "DeviceNotRegistered"
}

// receiptResponse is the full Expo getReceipts response body.
type receiptResponse struct {
	Data map[string]Receipt `json:"data"`
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

// Send fires a batch of push messages to the Expo Push API.
// Expo accepts up to 100 messages per request; callers must chunk larger batches.
// The returned slice is parallel to msgs: index i in the result corresponds to msgs[i].
func Send(ctx context.Context, msgs []Message) ([]Ticket, error) {
	if len(msgs) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(msgs)
	if err != nil {
		return nil, fmt.Errorf("push marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, expoSendEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("push send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("push API returned %d", resp.StatusCode)
	}

	var result ticketResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("push decode: %w", err)
	}
	return result.Data, nil
}

// SendToTokens sends the same title+body to all provided Expo push tokens.
// Empty tokens are skipped; the returned slice contains one entry per non-empty token
// in the same order they appear in the input.
func SendToTokens(ctx context.Context, tokens []string, title, body string, data any) ([]Ticket, error) {
	const batchSize = 100

	var msgs []Message
	for _, t := range tokens {
		if t == "" {
			continue
		}
		msgs = append(msgs, Message{
			To:    t,
			Title: title,
			Body:  body,
			Data:  data,
			Sound: "default",
		})
	}

	var all []Ticket
	for i := 0; i < len(msgs); i += batchSize {
		end := i + batchSize
		if end > len(msgs) {
			end = len(msgs)
		}
		batch, err := Send(ctx, msgs[i:end])
		if err != nil {
			return all, err
		}
		all = append(all, batch...)
	}
	return all, nil
}

// PollReceipts fetches delivery receipts for the given Expo ticket IDs.
// Returns a map from ticket ID to Receipt.
func PollReceipts(ctx context.Context, ticketIDs []string) (map[string]Receipt, error) {
	if len(ticketIDs) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(map[string][]string{"ids": ticketIDs})
	if err != nil {
		return nil, fmt.Errorf("receipts marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, expoReceiptsEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("receipts request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("receipts send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("receipts API returned %d", resp.StatusCode)
	}

	var result receiptResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("receipts decode: %w", err)
	}
	return result.Data, nil
}
