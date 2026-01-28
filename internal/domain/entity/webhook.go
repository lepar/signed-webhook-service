package entity

// WebhookRequest represents the incoming webhook payload
type WebhookRequest struct {
	User   string `json:"user"`
	Asset  string `json:"asset"`
	Amount string `json:"amount"`
}

// Validate validates the webhook request
func (w *WebhookRequest) Validate() error {
	if w.User == "" {
		return ErrMissingUser
	}
	if w.Asset == "" {
		return ErrMissingAsset
	}
	if w.Amount == "" {
		return ErrMissingAmount
	}
	return nil
}
