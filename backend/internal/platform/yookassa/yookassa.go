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
	PaymentMethodID string // set when save_payment_method=true and payment succeeded
}

// CreateParams describes a payment to create via CreatePayment.
type CreateParams struct {
	Amount            int // whole rubles
	Description       string
	ReturnURL         string
	Metadata          map[string]string
	SavePaymentMethod bool // set true for first subscription payment to enable auto-renewal
}

// RecurrentParams describes a server-initiated renewal payment (no redirect).
type RecurrentParams struct {
	Amount          int
	Description     string
	PaymentMethodID string
	Metadata        map[string]string
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
	Amount            ykAmount          `json:"amount"`
	Capture           bool              `json:"capture"`
	Confirmation      ykConfirmationReq `json:"confirmation"`
	Description       string            `json:"description,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	SavePaymentMethod bool              `json:"save_payment_method,omitempty"`
}

type ykRecurrentReq struct {
	Amount          ykAmount          `json:"amount"`
	Capture         bool              `json:"capture"`
	PaymentMethodID string            `json:"payment_method_id"`
	Description     string            `json:"description,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type ykPaymentResp struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Paid         bool   `json:"paid"`
	Confirmation struct {
		ConfirmationURL string `json:"confirmation_url"`
	} `json:"confirmation"`
	PaymentMethod struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"payment_method"`
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

func parsePayment(data []byte) (Payment, error) {
	var r ykPaymentResp
	if err := json.Unmarshal(data, &r); err != nil {
		return Payment{}, err
	}
	return Payment{
		ID:              r.ID,
		Status:          r.Status,
		Paid:            r.Paid,
		ConfirmationURL: r.Confirmation.ConfirmationURL,
		PaymentMethodID: r.PaymentMethod.ID,
	}, nil
}

// CreatePayment creates a redirect-confirmation payment and returns it.
func (c *Client) CreatePayment(ctx context.Context, p CreateParams) (Payment, error) {
	reqBody := ykCreateReq{
		Amount:            ykAmount{Value: fmt.Sprintf("%d.00", p.Amount), Currency: "RUB"},
		Capture:           true,
		Confirmation:      ykConfirmationReq{Type: "redirect", ReturnURL: p.ReturnURL},
		Description:       p.Description,
		Metadata:          p.Metadata,
		SavePaymentMethod: p.SavePaymentMethod,
	}
	data, status, err := c.do(ctx, http.MethodPost, "/payments", reqBody, idempotenceKey())
	if err != nil {
		return Payment{}, err
	}
	if status >= 300 {
		return Payment{}, fmt.Errorf("yookassa create payment: status %d: %s", status, string(data))
	}
	return parsePayment(data)
}

// CreateRecurrentPayment creates a server-side renewal payment using a saved
// payment method. The payment is captured automatically — no redirect needed.
func (c *Client) CreateRecurrentPayment(ctx context.Context, p RecurrentParams) (Payment, error) {
	if !c.Configured() {
		return Payment{}, fmt.Errorf("yookassa: not configured")
	}
	reqBody := ykRecurrentReq{
		Amount:          ykAmount{Value: fmt.Sprintf("%d.00", p.Amount), Currency: "RUB"},
		Capture:         true,
		PaymentMethodID: p.PaymentMethodID,
		Description:     p.Description,
		Metadata:        p.Metadata,
	}
	data, status, err := c.do(ctx, http.MethodPost, "/payments", reqBody, idempotenceKey())
	if err != nil {
		return Payment{}, err
	}
	if status >= 300 {
		return Payment{}, fmt.Errorf("yookassa recurrent payment: status %d: %s", status, string(data))
	}
	return parsePayment(data)
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
	return parsePayment(data)
}

// WebhookEvent is the notification YooKassa sends to our webhook URL.
type WebhookEvent struct {
	Event  string `json:"event"`
	Object struct {
		ID            string            `json:"id"`
		Status        string            `json:"status"`
		Paid          bool              `json:"paid"`
		Metadata      map[string]string `json:"metadata"`
		PaymentMethod struct {
			ID string `json:"id"`
		} `json:"payment_method"`
	} `json:"object"`
}

type ykRefundReq struct {
	Amount    ykAmount `json:"amount"`
	PaymentID string   `json:"payment_id"`
}

// RefundPayment issues a full refund for a succeeded payment.
func (c *Client) RefundPayment(ctx context.Context, ykPaymentID string, amountRub int) error {
	body := ykRefundReq{
		Amount:    ykAmount{Value: fmt.Sprintf("%d.00", amountRub), Currency: "RUB"},
		PaymentID: ykPaymentID,
	}
	data, status, err := c.do(ctx, http.MethodPost, "/refunds", body, idempotenceKey())
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("yookassa refund: status %d: %s", status, string(data))
	}
	return nil
}

// ParseWebhook decodes a webhook notification body.
func ParseWebhook(body []byte) (WebhookEvent, error) {
	var e WebhookEvent
	if err := json.Unmarshal(body, &e); err != nil {
		return WebhookEvent{}, err
	}
	return e, nil
}
