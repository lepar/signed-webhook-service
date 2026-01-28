package usecase

import (
	"context"

	"kii.com/internal/domain/entity"
	"kii.com/internal/domain/port"
)

// ProcessWebhookUseCase handles webhook processing
type ProcessWebhookUseCase struct {
	validator  port.WebhookValidator
	repository port.LedgerRepository
}

// NewProcessWebhookUseCase creates a new ProcessWebhookUseCase
func NewProcessWebhookUseCase(
	validator port.WebhookValidator,
	repository port.LedgerRepository,
) *ProcessWebhookUseCase {
	return &ProcessWebhookUseCase{
		validator:  validator,
		repository: repository,
	}
}

// ProcessWebhookRequest contains the request data for processing a webhook
type ProcessWebhookRequest struct {
	WebhookRequest *entity.WebhookRequest
	HTTPRequest    interface {
		Header() map[string][]string
		Body() []byte
	}
}

// Execute processes a webhook request
func (uc *ProcessWebhookUseCase) Execute(ctx context.Context, req ProcessWebhookRequest) error {
	// Validate webhook request entity
	if err := req.WebhookRequest.Validate(); err != nil {
		return err
	}

	// Create ledger entry
	entry := entity.LedgerEntry{
		User:   req.WebhookRequest.User,
		Asset:  req.WebhookRequest.Asset,
		Amount: req.WebhookRequest.Amount,
	}

	// Add to repository
	return uc.repository.AddEntry(ctx, entry)
}
