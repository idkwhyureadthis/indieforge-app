package yookassa

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBase = "https://api.yookassa.ru/v3"

// Client is a minimal YooKassa REST API client.
type Client struct {
	shopID    string
	secretKey string
	base      string
	http      *http.Client
}

// New builds a YooKassa client. An empty shopID/secretKey is valid — see
// Configured — and simply means payment creation will fail with a clear error.
func New(shopID, secretKey string) *Client {
	return &Client{
		shopID:    shopID,
		secretKey: secretKey,
		base:      defaultBase,
		http:      &http.Client{Timeout: 20 * time.Second},
	}
}

// Configured reports whether real credentials are present.
func (c *Client) Configured() bool { return c.shopID != "" && c.secretKey != "" }

// Payment is the subset of the YooKassa payment object we use.
type Payment struct {
	ID              string
	Status          string // pending | waiting_for_capture | succeeded | canceled
	Paid            bool
	ConfirmationURL string
}

// CreateParams describes a payment to create via CreatePayment.
type CreateParams struct {
	Amount      int // whole rubles
	Description string
	ReturnURL   string
	Metadata    map[string]string
}

type ykAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type ykConfirmationReq struct {
	Type      string `json:"type"`
	ReturnURL string `json:"return_url"`
}

type ykCreateReq struct {
	Amount       ykAmount          `json:"amount"`
	Capture      bool              `json:"capture"`
	Confirmation ykConfirmationReq `json:"confirmation"`
	Description  string            `json:"description,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type ykPaymentResp struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Paid         bool   `json:"paid"`
	Confirmation struct {
		ConfirmationURL string `json:"confirmation_url"`
	} `json:"confirmation"`
}

func idempotenceKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (c *Client) do(ctx context.Context, method, path string, body any, idem string) ([]byte, int, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		buf = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, buf)
	if err != nil {
		return nil, 0, err
	}
	req.SetBasicAuth(c.shopID, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	if idem != "" {
		req.Header.Set("Idempotence-Key", idem)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

// CreatePayment creates a redirect-confirmation payment and returns it.
func (c *Client) CreatePayment(ctx context.Context, p CreateParams) (Payment, error) {
	reqBody := ykCreateReq{
		Amount:       ykAmount{Value: fmt.Sprintf("%d.00", p.Amount), Currency: "RUB"},
		Capture:      true,
		Confirmation: ykConfirmationReq{Type: "redirect", ReturnURL: p.ReturnURL},
		Description:  p.Description,
		Metadata:     p.Metadata,
	}
	data, status, err := c.do(ctx, http.MethodPost, "/payments", reqBody, idempotenceKey())
	if err != nil {
		return Payment{}, err
	}
	if status >= 300 {
		return Payment{}, fmt.Errorf("yookassa create payment: status %d: %s", status, string(data))
	}
	var r ykPaymentResp
	if err := json.Unmarshal(data, &r); err != nil {
		return Payment{}, err
	}
	return Payment{ID: r.ID, Status: r.Status, Paid: r.Paid, ConfirmationURL: r.Confirmation.ConfirmationURL}, nil
}

// GetPayment fetches a payment's current state.
func (c *Client) GetPayment(ctx context.Context, id string) (Payment, error) {
	data, status, err := c.do(ctx, http.MethodGet, "/payments/"+id, nil, "")
	if err != nil {
		return Payment{}, err
	}
	if status >= 300 {
		return Payment{}, fmt.Errorf("yookassa get payment: status %d: %s", status, string(data))
	}
	var r ykPaymentResp
	if err := json.Unmarshal(data, &r); err != nil {
		return Payment{}, err
	}
	return Payment{ID: r.ID, Status: r.Status, Paid: r.Paid, ConfirmationURL: r.Confirmation.ConfirmationURL}, nil
}

// WebhookEvent is the notification YooKassa sends to our webhook URL.
type WebhookEvent struct {
	Event  string `json:"event"`
	Object struct {
		ID       string            `json:"id"`
		Status   string            `json:"status"`
		Paid     bool              `json:"paid"`
		Metadata map[string]string `json:"metadata"`
	} `json:"object"`
}

// ParseWebhook decodes a webhook notification body.
func ParseWebhook(body []byte) (WebhookEvent, error) {
	var e WebhookEvent
	if err := json.Unmarshal(body, &e); err != nil {
		return WebhookEvent{}, err
	}
	return e, nil
}
