package port

import (
	"context"
	"net/http"
)

// WebhookValidator is the port for webhook signature validation
type WebhookValidator interface {
	ValidateRequest(ctx context.Context, r *http.Request, body []byte) error
}
