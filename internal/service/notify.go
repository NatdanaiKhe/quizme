package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

// WebhookNotifier posts GenerateResult payloads to a webhook URL.
// It implements the Notifier seam.
type WebhookNotifier struct {
	WebhookURL string
	Client     *http.Client
}

// NewWebhookNotifier creates a notifier for webhookURL with the given timeout.
// A zero or negative timeout defaults to 5 seconds.
func NewWebhookNotifier(webhookURL string, timeout time.Duration) *WebhookNotifier {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &WebhookNotifier{
		WebhookURL: webhookURL,
		Client:     &http.Client{Timeout: timeout},
	}
}

type webhookPayload struct {
	BatchDate     string   `json:"batch_date"`
	Status        string   `json:"status"`
	QuestionCount int      `json:"question_count,omitempty"`
	Topics        []string `json:"topics,omitempty"`
	Error         string   `json:"error,omitempty"`
}

// Notify posts r to WebhookURL as JSON. It detaches from ctx cancellation so
// in-flight notifications survive handler-request shutdown. An empty
// WebhookURL is a no-op. Errors are logged with the URL host only and
// returned to the caller (the service discards them).
func (w *WebhookNotifier) Notify(ctx context.Context, r GenerateResult) error {
	ctx = context.WithoutCancel(ctx)
	if w.WebhookURL == "" {
		return nil
	}

	host := hostOnly(w.WebhookURL)

	p := webhookPayload{
		BatchDate: r.BatchDate.Format("2006-01-02"),
		Status:    r.Status,
	}
	if r.Status == "success" {
		p.QuestionCount = r.QuestionCount
		p.Topics = r.Topics
	} else if r.Error != nil {
		p.Error = r.Error.Error()
	}

	body, err := json.Marshal(p)
	if err != nil {
		log.Printf("webhook notify marshal error host=%s: %v", host, err)
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.WebhookURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("webhook notify request error host=%s: %v", host, err)
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.Client.Do(req)
	if err != nil {
		log.Printf("webhook notify POST failed host=%s: %v", host, err)
		return fmt.Errorf("webhook POST failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("webhook notify non-2xx host=%s status=%d", host, resp.StatusCode)
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func hostOnly(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "invalid-url"
	}
	return u.Host
}

var _ Notifier = (*WebhookNotifier)(nil)
